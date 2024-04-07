package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/cdp"
	cdpnetwork "github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	APP_NAME             = "cdpserver-230420-1658"
	SERVER_ADDRESS       = "127.0.0.1:3000"
	CDP_USER_AGENT       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36"
	RESULT_FILE_REPLACE  = true
	TASKS_QUEUE_SIZE     = 12
	WORKERS_POOL_SIZE    = 3
	FAILS_MAX_QUANTITY   = 10
	STATUS_SUCCESS       = "success"
	STATUS_BUSSY         = "bussy"
	STATUS_WRONG_REQUEST = "wrongRequest"
	STATUS_STOPPING      = "stopping"
)

type Task struct {
	Url      string `json:"url"`
	Selector string `json:"selector"`
	FileName string `json:"fileName"`
	TaskId   int    `json:"taskId,omitempty"`
}
type Response struct {
	Status     string          `json:"status"`
	TaskId     int             `json:"taskId,omitempty"`
	Processing *map[int]string `json:"processing,omitempty"`
}
type WorkerResult struct {
	WorkerNo int
	Error    error
}

var (
	checkCdpConnectionflag                    bool
	browserContext                            context.Context
	cancelBrowser                             context.CancelFunc
	browserHeaders                            cdpnetwork.Headers
	tasksChan                                 chan Task
	jsonServer                                *http.Server
	tasksQueue                                map[int]string
	tasksCounter                              int
	jsonBussy, jsonWrongRequest, jsonStopping []byte
	isStopping                                bool
	tasksQueueMutex                           chan bool
	exportResultMutex                         chan bool
	//mutex channel usage:
	//xxxMutex = make(chan bool, 1)
	//xxxMutex <- true //now channel is full until '<- xxxMutex' and another process has to wait
	//some action with xxx...
	//<- xxxMutex //now channel is free so another process can continue and update xxx
)

func exitApp() {
	exitCode := 0
	if r := recover(); r != nil {
		log.Println(r)
		//import "runtime/debug"
		//log.Println(r, string(debug.Stack()))
		exitCode = 1
	}
	log.Printf("exit code = %d\n", exitCode)
	os.Exit(exitCode)
}

func lockFile(isActive bool) error {
	fileName := "cdp-server.lock"
	_, err := os.Stat(fileName)
	if isActive {
		if err == nil {
			return fmt.Errorf("Server is running (lock-file %s was found)", fileName)
		}
		fp, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0766)
		if err != nil {
			return fmt.Errorf("Can not create %s: %v", fileName, err)
		}
		defer fp.Close()
		fp.WriteString("server is running")
	} else if err == nil {
		err = os.Remove(fileName)
		if err != nil {
			return fmt.Errorf("Can not remove %s: %v", fileName, err)
		}
	}
	return nil
}

func main() {
	log.SetFlags(log.Ltime) //|log.Lshortfile)
	defer exitApp()
	log.Printf("%s; Copyright Oloviannykov Gennadiy, 2023\n\n", APP_NAME)
	var (
		err            error
		sigTermChannel chan os.Signal
		//
		wr               WorkerResult
		workerResultChan chan WorkerResult
		workerResults    map[int]error
		//
		servErr         error
		servErrChan     chan error
		serverIsWorking bool
	)

	if err = lockFile(true); err != nil {
		panic(fmt.Errorf("lockFile: %v", err))
	}
	defer lockFile(false)

	//Ctrl-C / Ctrl-D channel
	sigTermChannel = make(chan os.Signal, 1)
	signal.Notify(sigTermChannel, syscall.SIGINT, syscall.SIGTERM)
	defer close(sigTermChannel)
	//
	if len(os.Args) > 1 && os.Args[1] == "check-connection" {
		checkCdpConnectionflag = true
	}
	if err = initBrowser(); err != nil {
		panic(fmt.Errorf("initBrowser: %v", err))
	}
	defer cancelBrowser()

	//sync channels
	tasksQueueMutex = make(chan bool, 1)
	exportResultMutex = make(chan bool, 1)
	defer func() {
		close(tasksQueueMutex)
		close(exportResultMutex)
	}()

	//result channels
	workerResultChan = make(chan WorkerResult, WORKERS_POOL_SIZE)
	defer close(workerResultChan)
	workerResults = make(map[int]error, WORKERS_POOL_SIZE)
	servErrChan = make(chan error, 1)
	defer close(servErrChan)
	//tasks common channel and registry
	tasksChan = make(chan Task, TASKS_QUEUE_SIZE)
	tasksQueue = make(map[int]string, TASKS_QUEUE_SIZE)
	tasksCounter = 0
	//precompiled JSON error responses
	jsonBussy, _ = json.Marshal(Response{Status: STATUS_BUSSY})
	jsonWrongRequest, _ = json.Marshal(Response{Status: STATUS_WRONG_REQUEST})
	jsonStopping, _ = json.Marshal(Response{Status: STATUS_STOPPING})
	//starting workers pool
	for workerNo := 1; workerNo <= WORKERS_POOL_SIZE; workerNo++ {
		log.Printf("Starting worker #%d\n", workerNo)
		go startWorker(workerNo, tasksChan, workerResultChan)
	}

	log.Println("Starting Webserver on", SERVER_ADDRESS)
	log.Println("POST /task - send tasks; /state - get queue state")
	log.Println("To stop press CTRL-C on Windows or CTRL-D on Linux")
	jsonServer = &http.Server{Addr: SERVER_ADDRESS}
	http.HandleFunc("/task", taskHandler)
	http.HandleFunc("/state", stateHandler)
	go func() {
		servErrChan <- jsonServer.ListenAndServe()
	}()
	serverIsWorking = true
	log.Printf("::::::::::::::::::::::::::::::::::::::::::::\n\n")

channelsListener:
	for loopsQty := 1; loopsQty <= WORKERS_POOL_SIZE+2; loopsQty++ {

		select {

		case <-sigTermChannel:
			log.Printf("****** STOP signal received ******\n\n")
			isStopping = true
			close(tasksChan)
			log.Printf("Tasks channel was closed\n")
			log.Printf("Waiting workers for finish\n")

		case servErr = <-servErrChan:
			log.Printf("****** server stopped (%v)\n", servErr)
			/*
			   if servErr!=nil && !errors.Is(servErr, http.ErrServerClosed) {
			       log.Printf("SERVER ERROR: %v\n", servErr)
			   }*/
			serverIsWorking = false
			isStopping = true
			close(tasksChan)
			log.Printf("Tasks channel was closed\n")
			log.Printf("Waiting workers for finish\n")

		case wr = <-workerResultChan:
			log.Printf("****** Worker #%d stopped (%v)\n", wr.WorkerNo, wr.Error)
			workerResults[wr.WorkerNo] = wr.Error
			if len(workerResults) == WORKERS_POOL_SIZE {
				log.Printf("All workers are closed\n")
				break channelsListener
			}

		} //end select

	} //end for

	log.Printf("\n:::::::::::::::: TERMINATING PROGRAM :::::::::::::::::\n")
	//close server if it didn't crash before
	if serverIsWorking {
		log.Println("5 seconds for shutdown server...")
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownRelease()
		if servErr = jsonServer.Shutdown(shutdownCtx); servErr != nil {
			log.Printf("Can not shutdown gracefully: %v.\nForsing to close", servErr)
			jsonServer.Close()
			// Close immediatelly terminates all active connections without waiting for them to be processed.
		}
	}
}

func initBrowser() error {
	var err error
	//avoid images downloading
	chromedp.Flag("blink-settings", "imagesEnabled=false")
	log.Printf("Creating browser context..\n")
	//params: parent context.Context, opts ...ContextOption
	//returns: context.Context, context.CancelFunc
	browserContext, cancelBrowser = chromedp.NewContext(
		context.Background(),
		//chromedp.WithDebugf(log.Printf),
		chromedp.WithErrorf(log.Printf),
	)
	/*
		NewContext creates a chromedp context from the parent context.
		The parent context's Allocator is inherited, defaulting to an ExecAllocator with DefaultExecAllocatorOptions.
		If the parent context contains an allocated Browser, the child context inherits it,
		and its on 1st Run creates a new tab on that browser. Otherwise - allocates a new browser.
		Cancelling the returned context will close a tab or an entire browser, depending on the logic described above.
		NewContext doesn't allocate nor start a browser; that happens the first time Run is used on the context.

			chromedp.Cancel:
				panics if called for the second time, as it tries to close a closed channel twice
				closes the browser gracefully and then waits for the process to end
				can hang, if the browser is not responsive
			chromedp.NewContext cancel
				can be called multiple times
				sends sigkill to the browser (through exec.CommandContext).
				sigkill sent to chrome running through a wrapper script does not kill the sub-processes so it cannot run clean up them
				never hangs, as it just kills the process
	*/
	if err = browserContext.Err(); err != nil {
		return fmt.Errorf("chromedp.NewContext: %v", err)
	}
	browserHeaders = cdpnetwork.Headers(map[string]interface{}{
		"user-agent": CDP_USER_AGENT,
	})
	if checkCdpConnectionflag {
		log.Printf("Testing navigation..\n")
		if err = browserTestNavigation(); err != nil {
			return fmt.Errorf("browserTestNavigation: %v", err)
		}
		log.Println("PASSED")
	}
	return nil
}

func browserTestNavigation() error {
	var (
		err            error
		testUrl, title string = "https://www.google.com/", ""
		pageLoaded     bool   = false
	)
	tabContext, cancelTab := chromedp.NewContext(browserContext)
	if err = tabContext.Err(); err != nil {
		return err
	}
	defer cancelTab()
	log.Printf("Navigating to %s..", testUrl)

	func() {
		timerContext, timerCancel := context.WithTimeout(tabContext, time.Duration(4)*time.Minute)
		defer timerCancel()
		err = chromedp.Run(timerContext, chromedp.Navigate(testUrl), chromedp.Title(&title))
		pageLoaded = true
	}()
	if err != nil {
		return err
	}
	if !pageLoaded {
		return fmt.Errorf("Page was not loaded in 5 minutes")
	}
	if title == "" {
		return fmt.Errorf("Page title not found")
	}
	return nil
}

func stateHandler(w http.ResponseWriter, r *http.Request) {
	if isStopping {
		w.Write(jsonStopping)
		return
	}

	tasksQueueMutex <- true
	qsize := len(tasksQueue)
	<-tasksQueueMutex
	if qsize == TASKS_QUEUE_SIZE {
		w.Write(jsonBussy)
		return
	}

	tasksQueueMutex <- true
	jsonBytes, _ := json.Marshal(Response{
		Status:     STATUS_SUCCESS,
		Processing: &tasksQueue,
	})
	<-tasksQueueMutex
	w.Write(jsonBytes)
}

func taskHandler(w http.ResponseWriter, r *http.Request) {
	//log.Printf("Requested %s %s\n", r.Method, r.URL.Path)
	switch r.Method {
	case "POST":
		if isStopping {
			w.Write(jsonStopping)
			return
		}
		d := json.NewDecoder(r.Body)
		task := Task{}
		err := d.Decode(&task)
		if err != nil {
			w.Write(jsonWrongRequest)
			return
		}
		task.TaskId = newTaskId(task.FileName)
		if task.TaskId == 0 {
			w.Write(jsonBussy)
			return
		}
		log.Printf("+ #%d (%s)\n", task.TaskId, task.FileName)
		tasksChan <- task
		jsonBytes, _ := json.Marshal(Response{
			Status: STATUS_SUCCESS,
			TaskId: task.TaskId,
		})
		w.Write(jsonBytes)
	default:
		w.Write(jsonWrongRequest)
	}
}

func newTaskId(caption string) int {
	n := 0
	tasksQueueMutex <- true
	if len(tasksQueue) < TASKS_QUEUE_SIZE {
		//taskIdMutex.Lock()
		//taskIdMutex is locked if another newTaskId() try to put value
		tasksCounter++
		tasksQueue[tasksCounter] = caption
		//now channel is free and another newTaskId() can put value
		//taskIdMutex.Unlock()
		n = tasksCounter
	}
	<-tasksQueueMutex
	return n
}
func removeTaskId(taskId int) {
	tasksQueueMutex <- true
	if _, ok := tasksQueue[taskId]; ok {
		//removeTaskMutex.Lock()
		delete(tasksQueue, taskId)
		//removeTaskMutex.Unlock()
	}
	<-tasksQueueMutex
}

func getCookieSettings() (targetDomainName string, cookiesForTheDomain []string) {
	//todo: load from file and divide into var-value pairs
	targetDomainName = "www.your-site-to-scrape.com"
	cookiesForTheDomain = []string{
		"cookie-key-1", "cookie-value-1",
		"cookie-key-2", "cookie-value-2",
		//...
	}
	return
}

func startWorker(workerNo int, tasksChan chan Task, workerResultChan chan WorkerResult) {
	workerResult := WorkerResult{WorkerNo: workerNo, Error: nil}
	defer func() {
		workerResultChan <- workerResult
	}()
	workerName := fmt.Sprintf("[worker-%d]", workerNo)
	var (
		err                  error
		tabContext           context.Context
		cancelTab            context.CancelFunc
		cdpActions           chromedp.Tasks
		maxTasksPerTab       int    = 50
		pageLoadWaitSelector string = `div.copyright-footer`
	)
	actionFuncSetCookies := chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		remoteDomain, cookies := getCookieSettings()
		expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
		for i := 0; i < len(cookies); i += 2 {
			err = cdpnetwork.SetCookie(cookies[i], cookies[i+1]).
				WithExpires(&expr).
				WithDomain(remoteDomain).
				WithHTTPOnly(false).
				Do(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})

	for { // tab reloads every maxTasksPerTab tasks

		//open new browser TAB
		tabContext, cancelTab = chromedp.NewContext(browserContext)
		if err = tabContext.Err(); err != nil {
			workerResult.Error = err
			return
		}

		cdpActions = chromedp.Tasks{

			actionFuncSetCookies,
			cdpnetwork.Enable(),
			cdpnetwork.SetExtraHTTPHeaders(browserHeaders),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var (
					task                                    Task
					taskName, viewPort, failCaption         string
					pageIsLoaded                            bool
					err                                     error
					failsQty                                int           = 0
					requestTimeoutMinutes, sleepTimeSeconds time.Duration = 4, 1
					longSleepTimeSeconds                    time.Duration = 10
					tasksCounter                            int           = 0
				)

				log.Printf("%s starting tasks loop\n", workerName)
				// range loop terminates once the chan is closed and buffer is empty,
				// otherwise it blocks if there is no value

				for task = range tasksChan {

					taskName = fmt.Sprintf("[task-%d %s]", task.TaskId, task.FileName)
					viewPort = ""
					pageIsLoaded = false
					tasksQueueMutex <- true
					tasksQueue[task.TaskId] += " " + workerName
					<-tasksQueueMutex
					//log.Printf("%s requesting %s\n", workerName, taskName)
					func() {
						timerContext, timerCancel := context.WithTimeout(
							ctx,
							time.Duration(requestTimeoutMinutes)*time.Minute,
						)
						defer timerCancel()
						chromedp.Navigate(task.Url).Do(timerContext)
						chromedp.WaitVisible(pageLoadWaitSelector, chromedp.ByQuery).Do(timerContext)
						//Page.stopLoading - Force the page stop all navigations and pending resource fetches.
						pageIsLoaded = true
						chromedp.OuterHTML(task.Selector, &viewPort, chromedp.ByQuery).Do(timerContext)
					}()
					if len(viewPort) == 0 {
						removeTaskId(task.TaskId)
						failsQty++
						failCaption = fmt.Sprintf("%s %s FAIL #%d", workerName, taskName, failsQty)
						if pageIsLoaded {
							log.Printf("%s: page was loaded but selector not appeared\n", failCaption)
						} else {
							log.Printf("%s: timeout reached during page loading\n", failCaption)
						}
						if failsQty == FAILS_MAX_QUANTITY {
							log.Printf("%s Fails max. quantity reached. Stopping worker", workerName)
							return fmt.Errorf("Fails max. quantity reached (%d times)", FAILS_MAX_QUANTITY)
						}
						time.Sleep(time.Duration(longSleepTimeSeconds) * time.Second)
						continue
					}
					failsQty = 0
					err = saveToFile(task.FileName, &viewPort)
					removeTaskId(task.TaskId)
					if err != nil {
						log.Printf("%s %s: %s\n", workerName, taskName, err)
						return fmt.Errorf("Can not save %s", task.FileName)
					}
					log.Printf("%s %s: DONE\n", workerName, taskName)

					if tasksCounter++; tasksCounter == maxTasksPerTab {
						log.Printf("%s halting for reload\n", workerName)
						return nil
					}

					time.Sleep(time.Duration(sleepTimeSeconds) * time.Second)

				} //tasks queue end

				//Page.close - Tries to close page, running its beforeunload hooks, if any.
				log.Printf("%s ended tasks loop\n", workerName)
				return nil
			}),
		} //end chromedp.Tasks

		err = chromedp.Run(tabContext, cdpActions)
		cancelTab()
		if err != nil || isStopping {
			break
		}
		time.Sleep(time.Duration(1) * time.Second)

	} //end tab reloads

	workerResult.Error = err
}

func saveToFile(filePath string, content *string) error {
	//exportResultMutex.Lock()
	//defer exportResultMutex.Unlock()
	exportResultMutex <- true
	defer func() {
		<-exportResultMutex
	}()
	//
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		if RESULT_FILE_REPLACE {
			err = os.Remove(filePath)
			if err != nil {
				return fmt.Errorf("Can not replace result file")
			}
		} else {
			return fmt.Errorf("File exists")
		}
	}
	fp, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer fp.Close()
	fp.WriteString(*content)
	return nil
}
