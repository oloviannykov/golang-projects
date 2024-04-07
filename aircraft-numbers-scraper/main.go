package main

import (
    "fmt"
    "io/ioutil"
    "bufio"
    "net/http"
    //"net/http/cookiejar"
    "net/url"
    "os"
    "regexp"
    "strings"
    "time"
)

var userAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/103.0.0.0 Safari/537.36"
var reservationURL string = "https://aircraft.faa.gov/e.gov/NN/reserve.aspx"
var faaClient *http.Client = nil

/*
func getResponseString(endpoint string) (string, error) {
    //req, err := http.NewRequest(http.MethodGet, endpoint, nil)
    // appending to existing GET query args
    //q := req.URL.Query()
    //q.Add("foo", "bar")
    // assign encoded query string to http request
    //req.URL.RawQuery = q.Encode()
    //resp, err := client.Do(req)
    resp, err := http.Get(endpoint)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    fmt.Println("Response status:", resp.Status)
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return "", err
    }
    return string(body), nil
}
*/

func postFormWithCookies(endPoint string, getParams, inputsMap, cookiesMap map[string]string) (int, string, error) {
    form := url.Values{} //map[string][]string
    for iName, iValue := range inputsMap {
        form.Add(iName, iValue)
    }
    //map[string][]string --> "...&name=Ava&friend=Jess"

    urlParams := url.Values{}
    for pName, pValue := range getParams {
        urlParams.Add(pName, pValue)
    }
    urlWithParams := endPoint + "?" + urlParams.Encode()
    req, err := http.NewRequest( http.MethodPost, urlWithParams, strings.NewReader(form.Encode()) )
    if err != nil {
        return 0, "", err
    }
    
    req.Header.Set("User-Agent", userAgent)
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    if len(cookiesMap) > 0 {
        for cName, cValue := range cookiesMap {
            req.AddCookie(&http.Cookie{Name: cName, Value: cValue})
        }
    }
    
    //jar, err := cookiejar.New(&cookiejar.Options{...})
    //Because the net/http/cookiejar package stores cookies in memory, the cookies will be destroyed 
    //once the program exits. If you want to persist the cookies, you can build your own cookie jar 
    //by implementing the http.CookieJar interface.
    //client := &http.Client{
    //    Jar: jar,
    //}
    //For control over proxies, TLS configuration, keep-alives, compression, and other settings, create a Transport:
    //tr := &http.Transport{
    //	MaxIdleConns:       10,
    //	IdleConnTimeout:    30 * time.Second,
    //	DisableCompression: true,
    //}
    response, err := faaClient.Do(req)
    //response, err := client.PostForm(endPoint, formFields)
    //resp, err := http.Post("https://httpbin.org/post", "application/json", bytes.NewBuffer(json_data))
    
    if err != nil {
        return 0, "", err
    }

    defer response.Body.Close()
    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return 0, "", err
    }
    return response.StatusCode, string(body), nil
}

func sendNNumbersReservationForm(nnumbers []string) (string, error) {
    getParams := map[string]string{"AspxAutoDetectCookieSupport":"1"}
    inputsMap := map[string]string{
        "_ctl0_content_ajaxtoolkit_HiddenField": ";;AjaxControlToolkit, Version=3.5.40412.0, Culture=neutral, PublicKeyToken=28f01b0e84b6d53e:en-US:1547e793-5b7e-48fe-8490-03a375b13a33:f2c8e708:de1feab2:720a52bf:f9cec9bc:4a2c8239",
        "__VIEWSTATE": "/wEPDwUKMTQxMzg0NDY5Nw8WAh4TVmFsaWRhdGVSZXF1ZXN0TW9kZQIBFgJmD2QWBgIDD2QWAmYPZBYCZg8WAh4EVGV4dAXHCDxkaXYgaWQ9J3ZOYXYnPjx1bD48bGk+PGxhYmVsIGNsYXNzPSJpc1RpdGxlICAiPkFpcmNyYWZ0PGJyLz5OLU51bWJlciBSZXNlcnZhdGlvbjwvbGFiZWw+PC9saT48L3VsPjx1bCBjbGFzcz0iSGFsZkJyZWFrIj48bGkgY2xhc3M9IkhhbGZCcmVhayI+JiMxNjA8L2xpPjwvdWw+PHVsPjxsaSBjbGFzcz0iIj48YSBjbGFzcz0iIiBocmVmPSJodHRwOi8vd3d3LmZhYS5nb3YvbGljZW5zZXNfY2VydGlmaWNhdGVzL2FpcmNyYWZ0X2NlcnRpZmljYXRpb24vYWlyY3JhZnRfcmVnaXN0cnkvc3BlY2lhbF9ubnVtYmVycy8iPkFib3V0IE4tTnVtYmVyIFJlc2VydmF0aW9uPC9hPjwvbGk+PC91bD48dWw+PGxpIGNsYXNzPSIiPjxhIGNsYXNzPSIiIGhyZWY9Imh0dHA6Ly9yZWdpc3RyeS5mYWEuZ292L2FpcmNyYWZ0aW5xdWlyeS8iPkFpcmNyYWZ0IElucXVpcnk8L2E+PC9saT48L3VsPjx1bD48bGkgY2xhc3M9IiI+PGEgY2xhc3M9IiIgaHJlZj0iaHR0cDovL3d3dy5mYWEuZ292L2xpY2Vuc2VzX2NlcnRpZmljYXRlcy9haXJjcmFmdF9jZXJ0aWZpY2F0aW9uL2FpcmNyYWZ0X3JlZ2lzdHJ5LyI+QWlyY3JhZnQgUmVnaXN0cmF0aW9uPC9hPjwvbGk+PC91bD48dWw+PGxpIGNsYXNzPSIiPjxhIGNsYXNzPSIiIGhyZWY9Imh0dHA6Ly9yZWdpc3RyeS5mYWEuZ292L2FpcmNyYWZ0aW5xdWlyeS9zZWFyY2gvbm51bWJlcmF2YWlsYWJpbGl0eWlucXVpcnkiPk4tTnVtYmVyIEF2YWlsYWJpbGl0eSBTZWFyY2g8L2E+PC9saT48L3VsPjx1bD48bGkgY2xhc3M9IiI+PGEgY2xhc3M9IiIgaHJlZj0iaHR0cDovL2FpcmNyYWZ0LmZhYS5nb3YvZS5nb3YvbnIiPlJlbmV3IE4tTnVtYmVyIFJlc2VydmF0aW9uPC9hPjwvbGk+PC91bD48dWw+PGxpIGNsYXNzPSIiPjxhIGNsYXNzPSIiIGhyZWY9Imh0dHA6Ly9yZWdpc3RyeS5mYWEuZ292L2FpcmNyYWZ0ZW1haWwvIj5Db250YWN0IEFpcmNyYWZ0IFJlZ2lzdHJhdGlvbjwvYT48L2xpPjwvdWw+PHVsPjxsaSBjbGFzcz0iIj48YSBjbGFzcz0iIiBocmVmPSJodHRwOi8vcmVnaXN0cnkuZmFhLmdvdi93ZWJhZG1pbmVtYWlsLyI+Q29udGFjdCBXZWIgQWRtaW5pc3RyYXRpb248L2E+PC9saT48L3VsPjwvZGl2PmQCBQ9kFgICAQ8WAh4Gb25sb2FkBSRzZXRGb2N1cyhfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIxKTsWAmYPZBYEZg9kFhICAQ8PZBYIHgdvbmZvY3VzBfcCYWR2YW5jZXRvPSdfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIyJzt2YXIgYSA9IG5ldyBBcnJheShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIxLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjIsX2N0bDBfY29udGVudF90eHROTnVtYmVyMyxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI0LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjUpO3NldGNvbG9ycyhhLCcjMDAwMDAwJywnI0ZGRkZGRicsJyM4Mzk5QjEnKTtzZXRCZ0NvbG9yKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LCcjOTk5OTk5Jyk7c2V0QmdDb2xvcihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgsJyM5OTk5OTknKTtkaXNhYmxlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlclN1ZmZpeCk7HgdvbmtleXVwBWp0b1VwcGVyKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEpO3NldFZhbHVlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsbWF0Y2hubihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIxKSk7HglvbmtleWRvd24FEEVudGVyTWVhbnNUYWIoKTseBm9uYmx1cgXlAnZhciBhID0gbmV3IEFycmF5KF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsX2N0bDBfY29udGVudF90eHROTnVtYmVyMixfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjQsX2N0bDBfY29udGVudF90eHROTnVtYmVyNSk7c2V0Y29sb3JzKGEsJyMwMDAwMDAnLCcjRkZGRkZGJywnIzgzOTlCMScpO3NoaWZ0ZHVwZXMoYSk7SWZWYWx1ZURpc2FibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyMSxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCx0cnVlKTtTZXRGb2N1c0lmVmFjYW50KF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsX2N0bDBfY29udGVudF90eHROTnVtYmVyQ291bnQpO2QCAg8PZBYIHwMFoAFhZHZhbmNldG89J19jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjMnO3NldEJvcmRlckNvbG9yKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjIsJyM4Mzk5QjEnKTtTZXRGb2N1c0lmVmFjYW50KF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsX2N0bDBfY29udGVudF90eHROTnVtYmVyMSk7HwQFanRvVXBwZXIoX2N0bDBfY29udGVudF90eHROTnVtYmVyMik7c2V0VmFsdWUoX2N0bDBfY29udGVudF90eHROTnVtYmVyMixtYXRjaG5uKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjIpKTsfBQUQRW50ZXJNZWFuc1RhYigpOx8GBawDdmFyIGEgPSBuZXcgQXJyYXkoX2N0bDBfY29udGVudF90eHROTnVtYmVyMSxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIyLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjMsX2N0bDBfY29udGVudF90eHROTnVtYmVyNCxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI1KTtzZXRjb2xvcnMoYSwnIzAwMDAwMCcsJyNGRkZGRkYnLCcjODM5OUIxJyk7c2hpZnRkdXBlcyhhKTtJZlZhbHVlRGlzYWJsZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIxLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LHRydWUpO0lmVmFsdWVEaXNhYmxlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsX2N0bDBfY29udGVudF90eHROTnVtYmVyU3VmZml4LGZhbHNlKTtTZXRGb2N1c0lmVmFjYW50KF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjIsX2N0bDBfY29udGVudF9idG5OZXh0KTtkAgMPD2QWCB8DBaABYWR2YW5jZXRvPSdfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI0JztzZXRCb3JkZXJDb2xvcihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzLCcjODM5OUIxJyk7U2V0Rm9jdXNJZlZhY2FudChfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIyLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjIpOx8EBWp0b1VwcGVyKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjMpO3NldFZhbHVlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjMsbWF0Y2hubihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzKSk7HwUFEEVudGVyTWVhbnNUYWIoKTsfBgWsA3ZhciBhID0gbmV3IEFycmF5KF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsX2N0bDBfY29udGVudF90eHROTnVtYmVyMixfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjQsX2N0bDBfY29udGVudF90eHROTnVtYmVyNSk7c2V0Y29sb3JzKGEsJyMwMDAwMDAnLCcjRkZGRkZGJywnIzgzOTlCMScpO3NoaWZ0ZHVwZXMoYSk7SWZWYWx1ZURpc2FibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyMSxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCx0cnVlKTtJZlZhbHVlRGlzYWJsZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIxLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlclN1ZmZpeCxmYWxzZSk7U2V0Rm9jdXNJZlZhY2FudChfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzLF9jdGwwX2NvbnRlbnRfYnRuTmV4dCk7ZAIEDw9kFggfAwWgAWFkdmFuY2V0bz0nX2N0bDBfY29udGVudF90eHROTnVtYmVyNSc7c2V0Qm9yZGVyQ29sb3IoX2N0bDBfY29udGVudF90eHROTnVtYmVyNCwnIzgzOTlCMScpO1NldEZvY3VzSWZWYWNhbnQoX2N0bDBfY29udGVudF90eHROTnVtYmVyMyxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzKTsfBAVqdG9VcHBlcihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI0KTtzZXRWYWx1ZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI0LG1hdGNobm4oX2N0bDBfY29udGVudF90eHROTnVtYmVyNCkpOx8FBRBFbnRlck1lYW5zVGFiKCk7HwYFrAN2YXIgYSA9IG5ldyBBcnJheShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIxLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjIsX2N0bDBfY29udGVudF90eHROTnVtYmVyMyxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI0LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjUpO3NldGNvbG9ycyhhLCcjMDAwMDAwJywnI0ZGRkZGRicsJyM4Mzk5QjEnKTtzaGlmdGR1cGVzKGEpO0lmVmFsdWVEaXNhYmxlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsX2N0bDBfY29udGVudF90eHROTnVtYmVyQ291bnQsdHJ1ZSk7SWZWYWx1ZURpc2FibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyMSxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgsZmFsc2UpO1NldEZvY3VzSWZWYWNhbnQoX2N0bDBfY29udGVudF90eHROTnVtYmVyNCxfY3RsMF9jb250ZW50X2J0bk5leHQpO2QCBQ8PZBYIHwMFpAFhZHZhbmNldG89J19jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50JztzZXRCb3JkZXJDb2xvcihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI1LCcjODM5OUIxJyk7U2V0Rm9jdXNJZlZhY2FudChfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI0LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjQpOx8EBWp0b1VwcGVyKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjUpO3NldFZhbHVlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjUsbWF0Y2hubihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI1KSk7HwUFEEVudGVyTWVhbnNUYWIoKTsfBgWKA3ZhciBhID0gbmV3IEFycmF5KF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsX2N0bDBfY29udGVudF90eHROTnVtYmVyMixfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjQsX2N0bDBfY29udGVudF90eHROTnVtYmVyNSk7c2V0Y29sb3JzKGEsJyMwMDAwMDAnLCcjRkZGRkZGJywnIzgzOTlCMScpO3NoaWZ0ZHVwZXMoYSk7SWZWYWx1ZURpc2FibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyMSxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCx0cnVlKTtJZlZhbHVlRGlzYWJsZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIxLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlclN1ZmZpeCxmYWxzZSk7c2V0Rm9jdXMoX2N0bDBfY29udGVudF9idG5OZXh0KTtkAgYPD2QWCB8DBZ8EYWR2YW5jZXRvPSdfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgnO2VuYWJsZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgpO1NldFZhbHVlSWZFcXVhbChfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCwnIyMnLCcnKTtzZXRGZ0NvbG9yKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LCcjMDAwMDAwJyk7c2V0QmdDb2xvcihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCwnI0ZBRkNGQycpO3NldEJnQ29sb3IoX2N0bDBfY29udGVudF90eHROTnVtYmVyU3VmZml4LCcjRkFGQ0ZDJyk7dmFyIGEgPSBuZXcgQXJyYXkoX2N0bDBfY29udGVudF90eHROTnVtYmVyMSxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIyLF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjMsX2N0bDBfY29udGVudF90eHROTnVtYmVyNCxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXI1KTtzZXRjb2xvcnMoYSwnIzAwMDAwMCcsJyNGRkZGRkYnLCcjODM5OUIxJyk7c2V0Y29sb3JzKGEsJyMwMDAwMDAnLCcjOTk5OTk5JywnIzgzOTlCMScpO2Rpc2FibGVhbGwoYSk7HwQFTnNldFZhbHVlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LG1hdGNobihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCkpOx8FBRBFbnRlck1lYW5zVGFiKCk7HwYF3gVTZXRDb2xvcklmVmFjYW50KF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LCcjMDAwMDAwJywnJyk7aWYgKGNoZWNrVmFsdWVSYW5nZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCwxLDEwLCdjb3VudF9ub3RpY2UnLGZhbHNlKSl7SWZWYWx1ZUVuYWJsZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgsZmFsc2UpO0lmVmFsdWVEaXNhYmxlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjUsdHJ1ZSk7SWZWYWx1ZURpc2FibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyQ291bnQsX2N0bDBfY29udGVudF90eHROTnVtYmVyNCx0cnVlKTtJZlZhbHVlRGlzYWJsZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJDb3VudCxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzLHRydWUpO0lmVmFsdWVEaXNhYmxlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjIsdHJ1ZSk7SWZWYWx1ZURpc2FibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyQ291bnQsX2N0bDBfY29udGVudF90eHROTnVtYmVyMSx0cnVlKTtTZXRGb2N1c0lmVmFjYW50KF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEpO1NldFZhbHVlSWZWYWNhbnQoX2N0bDBfY29udGVudF90eHROTnVtYmVyQ291bnQsJyMjJyk7fWQCBw8PZBYIHwMFImFkdmFuY2V0bz0nX2N0bDBfY29udGVudF9idG5OZXh0JzsfBAV5dG9VcHBlcihfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgpO3NldFZhbHVlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlclN1ZmZpeCxtYXRjaG5uKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlclN1ZmZpeCkpOx8FBRBFbnRlck1lYW5zVGFiKCk7HwYFrgRpZiAoY2hlY2tMZW5ndGhSYW5nZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgsMiwyLCdzdWZmaXhfbm90aWNlJyxmYWxzZSkpe0lmVmFsdWVFbmFibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyU3VmZml4LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlckNvdW50LGZhbHNlKTtJZlZhbHVlRGlzYWJsZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgsX2N0bDBfY29udGVudF90eHROTnVtYmVyNSxmYWxzZSk7SWZWYWx1ZURpc2FibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyU3VmZml4LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjQsZmFsc2UpO0lmVmFsdWVEaXNhYmxlKF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlclN1ZmZpeCxfY3RsMF9jb250ZW50X3R4dE5OdW1iZXIzLGZhbHNlKTtJZlZhbHVlRGlzYWJsZShfY3RsMF9jb250ZW50X3R4dE5OdW1iZXJTdWZmaXgsX2N0bDBfY29udGVudF90eHROTnVtYmVyMixmYWxzZSk7SWZWYWx1ZURpc2FibGUoX2N0bDBfY29udGVudF90eHROTnVtYmVyU3VmZml4LF9jdGwwX2NvbnRlbnRfdHh0Tk51bWJlcjEsZmFsc2UpO31kAggPD2QWBh4Hb25jbGljawUNUGxlYXNlV2FpdCgpOx8FBRNFbnRlck1lYW5zU3VibWl0KCk7HwMFE2FkdmFuY2V0bz0nc3VibWl0JztkAgkPD2QWBh8DBSJhZHZhbmNldG89J19jdGwwX2NvbnRlbnRfYnRuTmV4dCc7HwUFE0VudGVyTWVhbnNTdWJtaXQoKTsfBwUNUGxlYXNlV2FpdCgpO2QCAQ9kFggCAQ8QZGQWAGQCBg8QZGQWAGQCCA8QZGQWAGQCEA9kFgJmD2QWAgIBD2QWBgIFDw8WAh8BBQhWYWxpZGF0ZWRkAgcPDxYCHgdWaXNpYmxlaGRkAgkPDxYEHwEFCkluY29ycmVjdCEfCGhkZAIGD2QWAmYPZBYIZg9kFgJmD2QWAmYPFgIfAQXtATxkaXYgaWQ9J2Zvb3Rlcm1lbnUnPjxwIGNsYXNzPSJ0aXRsZSI+V2ViIFBvbGljaWVzPC9wPjx1bD48bGk+PGEgY2xhc3M9IiIgaHJlZj0iaHR0cDovL3d3dy5mYWEuZ292L3dlYl9wb2xpY2llcy8iPldlYiBQb2xpY2llcyBOb3RpY2VzPC9hPjwvbGk+PC91bD48dWw+PGxpPjxhIGNsYXNzPSIiIGhyZWY9Imh0dHA6Ly93d3cuZmFhLmdvdi9wcml2YWN5LyI+UHJpdmFjeSBQb2xpY3k8L2E+PC9saT48L3VsPjwvZGl2PmQCAQ9kFgJmD2QWAmYPFgIfAQWIBDxkaXYgaWQ9J2Zvb3Rlcm1lbnUnPjxwIGNsYXNzPSJ0aXRsZSI+R292ZXJubWVudCBTaXRlczwvcD48dWw+PGxpPjxhIGNsYXNzPSIiIGhyZWY9Imh0dHA6Ly93d3cuZG90Lmdvdi8iPkRPVC5nb3Y8L2E+PC9saT48L3VsPjx1bD48bGk+PGEgY2xhc3M9IiIgaHJlZj0iaHR0cDovL3d3dy51c2EuZ292LyI+VVNBLmdvdjwvYT48L2xpPjwvdWw+PHVsPjxsaT48YSBjbGFzcz0iIiBocmVmPSJodHRwOi8vd3d3LnBsYWlubGFuZ3VhZ2UuZ292LyI+UGxhaW5sYW5ndWFnZS5nb3Y8L2E+PC9saT48L3VsPjx1bD48bGk+PGEgY2xhc3M9IiIgaHJlZj0iaHR0cDovL3d3dy5yZWNvdmVyeS5nb3YvIj5SZWNvdmVyeS5nb3Y8L2E+PC9saT48L3VsPjx1bD48bGk+PGEgY2xhc3M9IiIgaHJlZj0iaHR0cDovL3d3dy5yZWd1bGF0aW9ucy5nb3YvIj5SZWd1bGF0aW9ucy5nb3Y8L2E+PC9saT48L3VsPjx1bD48bGk+PGEgY2xhc3M9IiIgaHJlZj0iaHR0cDovL3d3dy5kYXRhLmdvdi8iPkRhdGEuZ292PC9hPjwvbGk+PC91bD48L2Rpdj5kAgIPZBYCZg9kFgJmDxYCHwEFpwM8ZGl2IGlkPSdGQVFtZW51Jz48ZGl2IGNsYXNzPSJmYXEiPjxhIGhyZWY9Imh0dHA6Ly9mYWEuY3VzdGhlbHAuY29tLyI+RnJlcXVlbnRseSBBc2tlZCBRdWVzdGlvbnM8L2E+PC9kaXY+PGRpdiBjbGFzcz0iYWxsUXVlc3Rpb25zIj48YSBocmVmPSJodHRwOi8vd3d3LmZhYS5nb3YvbGljZW5zZXNfY2VydGlmaWNhdGVzL2FpcmNyYWZ0X2NlcnRpZmljYXRpb24vYWlyY3JhZnRfcmVnaXN0cnkvYWlyY3JhZnRfZmFxLyI+QWlyY3JhZnQgUmVnaXN0cmF0aW9uIEZBUSDCuzwvYT48L2Rpdj48ZGl2IGNsYXNzPSJhbGxRdWVzdGlvbnMiPjxhIGhyZWY9Imh0dHA6Ly9mYWEuY3VzdGhlbHAuY29tL2NnaS1iaW4vZmFhLmNmZy9waHAvZW5kdXNlci9zdGRfYWxwLnBocD9wX3NpZD11bHZ5dDlyaiI+QWxsIFF1ZXN0aW9ucyDCuzwvYT48L2Rpdj48L2Rpdj5kAgMPZBYCZg9kFgJmDxYCHwEF0AI8ZGl2IGlkPSdmb290ZXJtZW51Jz48cCBjbGFzcz0idGl0bGUiPkNvbnRhY3QgVXM8L3A+PHVsPjxsaT48YSBjbGFzcz0iIiBocmVmPSJodHRwOi8vd3d3LmZhYS5nb3YvYWJvdXQvb2ZmaWNlX29yZy9oZWFkcXVhcnRlcnNfb2ZmaWNlcy9haHIvY29udGFjdF91cy8iPkNvbnRhY3QgRkFBPC9hPjwvbGk+PC91bD48dWw+PGxpPjxhIGNsYXNzPSIiIGhyZWY9Imh0dHA6Ly93d3cub2lnLmRvdC5nb3YvaG90bGluZSI+T0lHIEhvdGxpbmU8L2E+PC9saT48L3VsPjx1bD48bGk+PGEgY2xhc3M9IiIgaHJlZj0iaHR0cDovL3d3dy5mYWEuZ292L2ZvaWEvIj5GT0lBPC9hPjwvbGk+PC91bD48L2Rpdj5kGAEFF19jdGwwOmNvbnRlbnQ6bXZSZXNlcnZlDw9kZmQTrB15BKydYAQ0jr9JIO7b1NBZXMtL9q4daHPCfDaYyw==",
        "__VIEWSTATEGENERATOR": "EB4A7FA2",
        "_ctl0:content:btnNext": "Proceed with Request",
        "_ctl0:content:txtNNumber1": "",
        "_ctl0:content:txtNNumber2": "",
        "_ctl0:content:txtNNumber3": "",
        "_ctl0:content:txtNNumber4": "",
        "_ctl0:content:txtNNumber5": "",
    }
    for i, n := range nnumbers {
        inputsMap[fmt.Sprintf("_ctl0:content:txtNNumber%d", i+1)] = n
    }
    cookiesMap := map[string]string{
        "AMCVS_AC781C8B53308D4B0A490D4D%40AdobeOrg": "1",
        "s_cc": "true",
        "ASP.NET_SessionId": "cutyozzdczh5xqr3agblpice",
        "cd_user_id": "18207f98bb717-0e64aae9774164-26021a51-144000-18207f98bb88e3",
        "QuantumMetricUserID": "d8637fc52dbd21ada93d9cc3bd7c249b",
        "_ga": "GA1.1.73862433.1657991149",
        "_ga_NQ5ZN114SB": "GS1.1.1657991149.1.1.1657991552.0",
        "_ga_XLYJSDG13C": "GS1.1.1657991149.1.1.1657991552.0",
        "AMCV_AC781C8B53308D4B0A490D4D%40AdobeOrg": "1099438348%7CMCIDTS%7C19192%7CMCMID%7C90004819335034751890213282549232944876%7CMCAAMLH-1658595069%7C7%7CMCAAMB-1658787289%7C6G1ynYcLPuiQxYZrsz_pkqfLG9yMXBpb2zX5dvJdYQJzPXImdj0y%7CMCOPTOUT-1658189689s%7CNONE%7CMCAID%7CNONE%7CMCSYNCSOP%7C411-19197%7CvVersion%7C2.1.0",
        "s_sq": "%5B%5BB%5D%5D",
    }   
    respCode, responseText, err := postFormWithCookies(reservationURL, getParams, inputsMap, cookiesMap)
    if err != nil {
        return "", err
    }
    if respCode != http.StatusOK {
        return responseText, fmt.Errorf("Got status %d '%s'", respCode, http.StatusText(respCode))
    }
    return responseText, nil
}


/*
Search for pairs N-Number + Status:
   [id="_ctl0_content_drptrResults__ctl1_lblNNumber">] [NNumber] [</span>]
   [id="_ctl0_content_drptrResults__ctl1_lblStatus">] [Status] [</span>]
  where Status must be 'Available for Request'
no available found if exists 'Sorry, none of the requested N-Numbers'
 *  */
func parseReservationPage(content *string) (map[string]bool, error) {
	numbersMap := make(map[string]bool, 5)
	var found bool
	//
	found, err := regexp.MatchString("none of the requested N-Numbers", *content)
	if err != nil {
            return nil, err
	}
	if found {
            return nil, nil
	}
	//
	rexNNumber := regexp.MustCompile(`id="_ctl0_content_drptrResults__ctl._lblNNumber">\s*([^<>]+)\s*</span>`)
	numberSubmatches := rexNNumber.FindAllStringSubmatch(*content, 5)
	//fmt.Printf("numberSubmatches: %v\n", numberSubmatches)
	if numberSubmatches == nil {
		return nil, fmt.Errorf("No numbers found")
	}
	//
	availabilityStatusText := "Available for Request"
	rexStatus := regexp.MustCompile(`id="_ctl0_content_drptrResults__ctl._lblStatus">\s*([^<>]+)\s*</span>`)
	statusSubmatches := rexStatus.FindAllStringSubmatch(*content, 5)
	//fmt.Printf("statusSubmatches: %v\n", statusSubmatches)
	//
	var number, statusText string
	for submatchIndex, submatchValue := range numberSubmatches {
		number = submatchValue[1]
		if len(statusSubmatches) < submatchIndex+1 {
			fmt.Printf("Status #%d for %s was not found", submatchIndex, number)
			continue
		}
		statusText = statusSubmatches[submatchIndex][1]
		//fmt.Printf("[%d] %s is %s\n", submatchIndex, number, statusText)
		numbersMap[number] = availabilityStatusText == statusText
	}
	return numbersMap, nil
}

func funcError(funcName string, param1 interface{}, err error) error {
   return fmt.Errorf("%v in %s(%v, ...)", err, funcName, param1)
}

func processNNumbersReservation(numbersBuffer []string) (int, error) {
    defer showExecutionTime("processNNumbersReservation")()
    resp, err := sendNNumbersReservationForm(numbersBuffer)
    if err != nil {
        return 0, funcError("sendNNumbersReservationForm", numbersBuffer, err)
    }
    numbersStatusMap, err := parseReservationPage(&resp)
    if err != nil {
        return 0, funcError("parseReservationPage", "...", err)
    }
    boolStatus, ok, availableNumbers := false, false, []string{}
    for _, strNumber := range numbersBuffer {
        //todo: save to result file
        boolStatus, ok = numbersStatusMap[strNumber]
        if ok && boolStatus {
            availableNumbers = append(availableNumbers, strNumber)
        }
    }
    if len(availableNumbers)>0 {
        fmt.Printf(" available %v\n", availableNumbers)
    } else {
        fmt.Println(" none available")
    }
    return len(availableNumbers), nil
}

func fileExists(fileName string) bool {
    info, err := os.Stat(fileName)
    if err!=nil {
        if !os.IsNotExist(err) {
           fmt.Printf("fileExists(%s) error: %v\n", fileName, err)
        }
        return false
    }
    return !info.IsDir()
}


func showExecutionTime(funcName string) func() {
    startTime := time.Now()
    return func() {
        fmt.Printf("%s - execution time %s\n", funcName, time.Since(startTime))
    }
}

////////////////

func main() {
    //useful tip: os.Exit skips the execution of deferred function.

    if len(os.Args)<3 {
        fmt.Printf("Usage: (this program) (input file name) (result-file-name)")
        os.Exit(1)
    }
    //get path from command line args
    inputPath, resultPath := os.Args[1], os.Args[2]
    fmt.Printf("Settings: input from %s; save to %s\n", inputPath, resultPath)

    if ! fileExists(inputPath) {
        fmt.Println(inputPath, " not found")
        os.Exit(2)
    }
    inputHandler, err := os.Open(inputPath)
    if err != nil {
        fmt.Println(err)
        os.Exit(3)
    }
    defer showExecutionTime("main")()

    inputScanner := bufio.NewScanner(inputHandler)
    inputScanner.Split(bufio.ScanLines)
    numbersBuffer := make([]string, 0, 5)
    numbersQty, availableQty := 0, 0
    
    faaClient = &http.Client{
        //Transport: tr
        Timeout: 30 * time.Second,
        //CheckRedirect: redirectPolicyFunc,
    }

    //read line by line, output as string and add to the slice
    for inputScanner.Scan() {
        //fmt.Println(inputScanner.Text())
	numbersBuffer = append(numbersBuffer, inputScanner.Text())
        if len(numbersBuffer)==5 {
            fmt.Printf("\nprocessing %v... ", numbersBuffer)
            availQty, err := processNNumbersReservation(numbersBuffer)
            if err != nil {
                fmt.Printf("ERROR: %v\n", err)
                numbersBuffer = []string{}
                break
            }
            numbersQty += 5
            availableQty += availQty
            numbersBuffer = []string{}
        }
    }
    if len(numbersBuffer)>0 {
        fmt.Printf("\nprocessing %v... ", numbersBuffer)
        availQty, err := processNNumbersReservation(numbersBuffer)
        if err != nil {
            fmt.Printf("ERROR: %v\n", err)
        }
        numbersQty += len(numbersBuffer)
        availableQty += availQty
    }
	
    //end reading
    inputHandler.Close()
    fmt.Printf("\nFrom %d numbers available %d\n", numbersQty, availableQty)
    //Processed 88 numbers
    //main - execution time 24.3618131s
    
    //todo: crosscompile for linux
    //todo: use go-routines and channals
    //todo: save each routine's result to individual 'processing-*.txt' file, rename to 'result-*.txt' before end
}
