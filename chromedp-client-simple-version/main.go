package main

import (
	"bufio"
	"strings"
	"context"
	"encoding/csv"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"
	"runtime/debug"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const APP_NAME = "cdpclient-230318-1014"

const RESULT_FILE_REPLACE = true
const (
	taskUrlIdx int = iota
	taskBlockSelectorIdx
	taskResultFileIdx
)
type TaskRecord struct {
	url      string
	selector string
	fileName string
}
var (
    tasksCSV, pageUrl, fragmentSelector, resultFileName string
    tasks []TaskRecord
    tasksCsvHeader []string = []string{"url", "sel", "res"}
    logFile *os.File
)


func initApp()  {
    var err error
    logName := fmt.Sprintf("%s %s pid%d.log", APP_NAME, time.Now().Format("2006-1-2-15"), os.Getpid())
    logFile, err = os.OpenFile(logName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0655)
    if err!=nil {
        panic(err)
    }
    log.SetOutput(logFile)
    log.SetFlags(log.Ltime|log.Lshortfile)
}

func exitApp() {
    exitCode := 0
    if r:=recover(); r!=nil {
		//fmt.Println("ERROR: ", r)
		//debug.PrintStack()
		//log.Printf("%v\n", string(debug.Stack()))
		log.Println(r, string(debug.Stack()))
		exitCode = 1
    }
    log.Printf("exit code = %d\n", exitCode)
    logFile.Close()
    os.Exit(exitCode)
}

func loadArgs() {
    flag.StringVar(&tasksCSV, "tasks", "", "tasks CSV file name; header 'url,sel,res'; don't combine with other flags")
    flag.StringVar(&pageUrl, "url", "", "page URL if tasks CSV not set")
    flag.StringVar(&fragmentSelector, "sel", "", "CSS selector")
    flag.StringVar(&resultFileName, "res", "", "file name to save result")
    flag.Parse()
    //Flag parsing stops just before the first non-flag argument ("-" is a non-flag argument) or after the terminator "--".
    //Integer flags accept 1234, 0664, 0x1234 and may be negative. Boolean flags may be:
    //1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False
    if tasksCSV=="" && (pageUrl=="" || fragmentSelector=="" || resultFileName=="") {
        fmt.Println(APP_NAME)
        flag.Usage()
        os.Exit(2)
        //defer funcs are not executed
    }
    //args after 1st not '-xx' or '--xx' flag
    //fmt.Println("tail:", flag.Args())
    //`app.exe -f="qwerty" --z abc 123 -x="asdf"` --> tail: abc 123 -x="asdf" 
}

func main() {
        //avoid log.Fatal() and log.Fatalln():
        //It’s just a shortcut for log.Print(v); os.Exit(1) - it exits immediately without running defer
        loadArgs()
        initApp()
        defer exitApp()
        //now we can use panic-recover pattern
        var err error
        if tasksCSV == "" {
                if ! strings.HasPrefix(pageUrl, "http") {
                    pageUrl, err = b64decode(pageUrl)
                    if(err!=nil) {
                        panic(fmt.Sprintf("failed to decode URL %v\nURL = %s", err, pageUrl))
                    }
                }
                
	} else {
		if _, err = os.Stat(tasksCSV); os.IsNotExist(err) {
			panic("Tasks CSV file not found")
		}
	}

//	  func NewContext(parent context.Context, opts ...ContextOption) (context.Context, context.CancelFunc)
//        NewContext creates a chromedp context from the parent context.
//        The parent context's Allocator is inherited, defaulting to an ExecAllocator with DefaultExecAllocatorOptions.
//>> 	  If the parent context contains an allocated Browser, the child context inherits it, and its first Run creates a new tab on that browser.
//>>      Otherwise, its first Run will allocate a new browser.
//        Cancelling the returned context will close a tab or an entire browser, depending on the logic described above.
//        Note that NewContext doesn't allocate nor start a browser; that happens the first time Run is used on the context.
	
	cdpContext, cancelCDP := chromedp.NewContext(
		context.Background(),
		//chromedp.WithDebugf(log.Printf),
		chromedp.WithErrorf(log.Printf),
	)
	defer cancelCDP()
	chromedp.Flag("blink-settings", "imagesEnabled=false")

	if tasksCSV == "" {
		tasks = append(tasks, TaskRecord{
			url:      pageUrl,
			selector: fragmentSelector,
			fileName: resultFileName,
		})
	} else {
		//fmt.Printf("Loading tasks from %s\n", tasksCSV)
		tasks, err = loadTasksFromCSV(tasksCSV)
		if err != nil {
			panic(err)
		}
	}

	log.Printf("Got %d tasks\n", len(tasks))
        err = processTasks(cdpContext)
        if err!=nil {
            panic(err)
        }
}

/*
func b64encode(textToEncode string) string {
    return base64.StdEncoding.EncodeToString([]byte(textToEncode))
}*/
func b64decode(textInBase64 string) (string, error) {
    //example: "aGVsbG8gZnJvbSBnb3NhbXBsZXMuZGV2IGJhc2U2NCBlbmNvZGluZyBleGFtcGxlIQ=="
    decodedBytes, err := base64.StdEncoding.DecodeString(textInBase64)
    if err != nil {
        return "", err
    }
    return string(decodedBytes), nil
}

func saveToFile(filePath string, content *string) error {
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

func loadTasksFromCSV(fileName string) ([]TaskRecord, error) {
	var fp *os.File
	var fReader *bufio.Reader
	var csvReader *csv.Reader
	var err error
	var r []string
	var tasks []TaskRecord
	fp, err = os.Open(fileName)
	if err != nil {
		return tasks, err
	}
	defer fp.Close()

	fReader = bufio.NewReader(fp)
	csvReader = csv.NewReader(fReader)
	csvReader.Comma = ','
	//read header
	_, err = csvReader.Read()
	if err == io.EOF {
		return tasks, fmt.Errorf("Header not found")
	}

	for {
		r, err = csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return tasks, err
		}
		if len(r) < 3 {
			continue
		}
		tasks = append(tasks, TaskRecord{
			url:      r[taskUrlIdx],
			selector: r[taskBlockSelectorIdx],
			fileName: r[taskResultFileIdx],
		})
	}
	return tasks, nil
}

func getRequestSettings() (string, string, []string) {
	//todo: load from file and divide into var-value pairs
	return "www.your-site-to-scrape.com",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/110.0.0.0 Safari/537.36",
		[]string{
			"cookie-key-1", "cookie-value-1",
			"cookie-key-2", "cookie-value-2",
			//...
		}
}

func processTasks(cdpContext context.Context) error {
    //cdpContext, cancelTimeout := context.WithTimeout(cdpContext, 60*time.Second)
	//defer cancelTimeout()

	remoteDomain, userAgent, cookies := getRequestSettings()
	extraHeaders := network.Headers(map[string]interface{}{
		"user-agent": userAgent,
	})

	//var nodes []*chromedp.Node
	cdpActions := chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			// create cookie expiration
			expr := cdp.TimeSinceEpoch(time.Now().Add(180 * 24 * time.Hour))
			// add cookies to chrome
			for i := 0; i < len(cookies); i += 2 {
				err = network.SetCookie(cookies[i], cookies[i+1]).
					WithExpires(&expr).
					WithDomain(remoteDomain).
					WithHTTPOnly(false).
					Do(ctx)
				if err != nil {
					return err
				}
			}
			return nil
		}),
		network.Enable(),
		network.SetExtraHTTPHeaders(extraHeaders),
                chromedp.ActionFunc(func(ctx context.Context) error {
                    var (
                        viewPort string
                        err error
                        taskDuration time.Duration
                    )
                        
                    for i, tsk := range tasks {
                        if tsk.url == "" || tsk.selector == "" || tsk.fileName == "" {
                            log.Printf("!! missing value in task %d {url:%v, selector:%v, file:%v}\n", i, tsk.url, tsk.selector, tsk.fileName)
                            continue
                        }
                        log.Printf("task %d: %s\n", i, tsk.url)
                        //
                        viewPort = ""
                        taskDuration = 4
                        func() {
                            timerContext, timerCancel := context.WithTimeout(ctx, time.Duration(taskDuration)*time.Minute)
                            defer timerCancel()
                            chromedp.Navigate(tsk.url).Do(timerContext)
                            chromedp.WaitVisible(`div.copyright-footer`, chromedp.ByQuery).Do(timerContext)
                            //default By func isn't ByQuery, but BySearch - which is pretty much a plaintext search over the page's HTML
                            //chromedp.Nodes(`div.some-class`, &nodes)
                            chromedp.OuterHTML(tsk.selector, &viewPort, chromedp.ByQuery).Do(timerContext)
                        }()
                        if len(viewPort)>0 {
                            log.Printf("loaded %d bytes to viewPort\n", len(viewPort))
                        } else {
                            log.Println("NOTHING LOADED - timeout reached or selector is wrong")
                            continue
                        }
                        //
                        err = saveToFile(tsk.fileName, &viewPort)
                        if err != nil {
                            log.Printf("-- FAILED SAVING '%s': %s\n", tsk.fileName, err)
                            continue
                        }
                        log.Printf("-- saved to '%s'\n", tsk.fileName)
                        time.Sleep(time.Duration(1) * time.Second)
                    }
                    return nil
                }),
	}
	if err := chromedp.Run(cdpContext, cdpActions); err != nil {
		log.Printf("chromedp.Run() failed\n")
		return err
	}
	return nil
}
