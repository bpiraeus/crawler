package main


import (
	
	"exp/html"
	"log"
	"time"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
//	"io"
//	"bufio"
	"os"
	"regexp"
	"strings"
	"fmt"
	"flag"
//	"crypto/tls"
	
)

var page string
var numc int
var llnw bool
var cache4 map[string]string
var cache6 map[string]string

type Worker struct {

        id      int
	client	*httputil.ClientConn
	last	string
        in      chan *Work
        out     chan *Work
        control chan int
}

// struct to contain information about the work to be done
type Work struct {

        page     string
}

func init() {

	flag.StringVar(&page, "u", "http://www.limelightnetworks.com/", "Page to test")
	flag.IntVar(&numc, "c", 3, "number of connections to use")
	flag.BoolVar(&llnw, "l", false, "enable llnw debug header")
}

func main() {

	flag.Parse()

	// change our logger to not output timestamps
	log.SetFlags(0)

//	mainStart := time.Now()

	// prep some channels
	//in := make(chan *Work, 1000)
	//out := make(chan *Work, 1000)
	//control := make(chan int)
//	client := httputil.NewClientConn(nil, nil)
//	last := ""

	// prep our maps
	cache4 = make(map[string]string, 0)
	cache6 = make(map[string]string, 0)

	// prep a variable for short term storage of a string
	var npage string

        // initialize our workers
//        for n := 0; n < numc; n++ {
//                w := NewWorker(n, last, client, in, out, control)
//                go w.StartWorker()
//        }

	u, uerr := url.Parse(page)
	if uerr != nil {
		log.Fatalf("Could not parse %s : %s", page, uerr)
	}
	
	ipaddr, timer, _, cerr := CacheLookup(u.Host)
	if cerr != nil {
		log.Fatalf("Could not lookup %s : %s", u.Host, cerr)
	}
	log.Printf("%s resolved to %s in %dms", u.Host, ipaddr, timer / 1000000)

	resp, err := http.Get(page)
	if err != nil {
		log.Printf("Could not get base page: %s : %s", page, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	document := html.NewTokenizer(resp.Body)
	document.AllowCDATA(true)
	if err != nil {
		log.Fatalf("Could not parse html body")
	}

	var hparse func(*html.Tokenizer)
	token := document.Next()
	hparse = func(*html.Tokenizer) {
		log.Printf("Entering hparse function")
		isScript := false
		for token != html.ErrorToken {
			x,y := document.TagName()
			text := document.Text()
			log.Printf("DOCTEXT: (%s) %s", string(x), string(text))
			if string(x) == "script" {
				isScript = true
			}
			if string(x) == "" && isScript {
				npage = ParseScript(text)
				isScript = false
			}
			if y == true {
				k,v,m := document.TagAttr()
				if m == false {
					if (string(k) == "src" || string(k) == "style") {
						npage = dispatch(v, u.Scheme)
					}
				}
				manual := false
				for m != false {
					switch string(v) {
					case "stylesheet", "text/javascript":
						manual = true
					}
					if (string(k) == "src" || string(k) == "style") {
						npage = dispatch(v, u.Scheme)
					}
					if (manual == true && string(k) == "href") {
						npage = dispatch(v, u.Scheme)
					}
					log.Printf("%s -> %s : %t : %t", string(k), string(v), m, manual)
					k,v,m = document.TagAttr()
					log.Printf("(2)%s -> %s : %t : %t", string(k), string(v), m, manual)
					if string(k) == "src" {
						npage = dispatch(v, u.Scheme)
					}
				}
				if npage != "" {
					log.Printf("Fetch-> %s", npage)
					npage = ""
				}
			}
			switch string(x) {
			case "div":
				innerdocument := html.NewTokenizerFragment(resp.Body, string(x))
				innerdocument.AllowCDATA(true)
				hparse(innerdocument)
			}
		token = document.Next()
		}
	}
	hparse(document)

}

func dispatch(val []byte, scheme string) (newpage string) {
	log.Printf("String Val: %s", string(val))

	isHttp, _ := regexp.Compile(`^http`)
	isShort, _ := regexp.Compile(`^//`)
	isRelative, _ := regexp.Compile(`^/`)
	isBground, _ := regexp.Compile(`^background:url\((.*)\)`)

	if isBground.Match(val) {
		nmatch := isBground.FindSubmatch(val)
		val = nmatch[1]
	}

	if isHttp.Match(val) {
		newpage = string(val)
		log.Printf("Matched isHttp: %s", string(val))
		return
	} else if isShort.Match(val) {
		newpage = fmt.Sprintf("%s:%s", scheme, string(val))
		log.Printf("Matched isShort: %s", string(val))
		return
	} else if isRelative.Match(val) {
		newpage = fmt.Sprintf("%s%s", page, strings.TrimLeft(string(val), "/"))
		log.Printf("Matched isRelative: %s", string(val))
		return
	} 

	log.Printf("No Match: %s", string(val))
	return
}
	
/*
		t := html.NewTokenizer(resp.Body)
		
		// disposable var
		var newpage string

		for {
			nt := t.Next()
			if nt == html.ErrorToken {
				if t.Err() == io.EOF {
					break
				} else {
					log.Printf("Error occurred: %s", t.Err())
					break
				}
			}
			switch nt {
				case html.StartTagToken:
				manual := ""
				key, val, _ := t.TagAttr()
				log.Printf("Key = %s -> %s", string(key), string(val))

				if string(key) == "style" {
					//log.Printf("Style Found: %s", string(val))
					if isBground.Match(val) {
						nmatch := isBground.FindSubmatch(val)
						manual = "yes"
						val = nmatch[1]
					}
				} else if string(key) == "rel" {
					if string(val) == "stylesheet" {
						tt := t.Token()
						manual = "yes"
						val = []byte(tt.Attr[1].Val)
						//log.Printf("StyleSheet Found: %s", string(val))
					}
				}
				if (string(key) == "src" || manual == "yes")  {
					//log.Printf("Token found: %s -> %s : %s", nt.String(), key, val)
					if isHttp.Match(val) {
						newpage = string(val)
						CacheLookup(newpage)
						in <- &Work{page: newpage}
					} else if isShort.Match(val) {
						newpage = fmt.Sprintf("http:%s", string(val))
						CacheLookup(newpage)
						in <- &Work{page: newpage}
					} else if isRelative.Match(val) {
						newpage = fmt.Sprintf("%s%s", page, strings.TrimLeft(string(val), "/"))
						CacheLookup(newpage)
						in <- &Work{page: newpage}
					}
				}
			}
		}
		close(in)
	}()

	go func() {
		for n := 0; n < numc; n++ {
			<-control
		}
		close(out)
	}()

	for _ = range out {
//		log.Printf("rolling")
	}

	mainStop := time.Now()
	mainElapsed := mainStop.Sub(mainStart) / 1000000
	log.Printf("Total Time: %d ms", mainElapsed)

}
						
// safe wrapper for testHost so we can recover from gross errors which would otherwise panic() the system and cause an early exit
func (w *Worker) safeTestObject(work *Work) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%s broke due to : %s\n", work.page, err)
		}
	}()
	w.TestObject(work)
}
	
// liberally stolen from http_check because we want to get lots of data and sometimes code reuse is good
// we have to modify it a bit, but otherwise it's got all the pieces we want
func ( w *Worker) TestObject(work *Work) {

	// pre-define myConn/tlsConn/client
	var myConn net.Conn
	var tlsConn *tls.Conn

	// variable for our address, a variable for our port (default to 80), and a variable to determine if it's ssl, and finally one if we need a new client connection or not
	var address string = ""
	var port = 80
	var ssl bool
	var needclient bool

	// create a buffer we'll use later for reading client body
	buffer := make([]byte, 1024)
	
	// build our request
	request, err := http.NewRequest("GET", work.page, nil)
	if err != nil {
		log.Printf("Worker %d: Failed to build request for %s : %s\n", w.id, work.page, err)
		return
	}

	// if llnw debug is enabled add our debug header
	if llnw {
		request.Header.Add("X-LDebug", "Yes")
	}

	u, uerr := url.Parse(work.page)
	if uerr != nil {
		log.Printf("Worker %d: Unable to lookup %s : %s", w.id, u.Host, uerr)
	}

	// is this an ssl connection?
	if u.Scheme == "https" {
		ssl = true
		port = 443
	}

	// do we have a pre-existing cached connection?
	if w.client != nil {
	//	log.Printf("Worker %d: found existing client, testing for reuse %s vs %s\n", w.id, w.last, u.Host)
		if w.last == u.Host {
			needclient = false
		} else {
			needclient = true
		}
	} else {
		needclient = true
	}
	w.last = u.Host
	
	// set a timer for the connection
	timestart := time.Now()

	ipaddr, iplookup, cached, ierr := CacheLookup(u.Host)
	if ierr != nil {
		log.Printf("Worker %d: Failed to lookup %s : %s", w.id, u.Host, ierr)
		return
	}

	// build our address string
	address = fmt.Sprintf("%s:%d", ipaddr, port)

	if needclient {
		// since we want some very low level access to bits and pieces, we're going to have to use tcp dial vs the native http client
		// create a net.conn with a dial timeout of 30 seconds to be very generous
		myConn, err = net.DialTimeout("tcp", address, 30 * time.Second)
		
		if err != nil {
			log.Printf("Worker %d:  Could not connect to %s : %s\n", w.id, address, err)
			return
		}

	}
	// get a time reading on how long it took to connect to the socket
	timestop := time.Now()
	tcpConnect := (timestop.Sub(timestart))

	if needclient {
		// need to add some deadlines so we don't sit around indefintely - 5s is more than sufficient
		myConn.SetDeadline(time.Now().Add(time.Duration(5 * time.Second)))
	}

	if needclient {
		// if we're an ssl connection, we need a few extra steps here
		if ssl {

			// default to requiring certificate validation
			tlsConfig := tls.Config{InsecureSkipVerify: false}

			// create a real tls connection
			tlsConn = tls.Client(myConn, &tlsConfig)

			// do our SSL negotiation
			err = tlsConn.Handshake()
			if err != nil {
				log.Printf("Worker %d: Could not negotiate tls handshake on %s : %s\n", w.id, address, err)
				return
			}
		}
	}
	
	// get a time reading on how long it took to negotiate ssl
	timestop = time.Now()
	sslHandshake := (timestop.Sub(timestart))

	if needclient {
		// convert to an http connection
		if ssl {
			w.client = httputil.NewProxyClientConn(tlsConn, nil)
		} else {
			w.client = httputil.NewProxyClientConn(myConn, nil)
		}
	}

	// write our request to the socket
	err = w.client.Write(request)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			log.Printf("Worker %d: Our persistent connection was wiped out: %s\n", w.id, err)
			return
		}
		log.Printf("Worker %d: Error writing request : %s\n", w.id, err)
		return
	}

	// store our size
	hsize := 0
	bsize := 0

	// read our response headers
	response, err := w.client.Read(request)
	if err != nil {
		if err != httputil.ErrPersistEOF {
			log.Printf("Worker %d: Error reading response : %s\n", w.id, err)
		}
		return
	}

	// did we get a response?
	if len(response.Header) == 0 {
		log.Printf("Worker %d: 0 length response, something probably broke", w.id)
		return
	}

	hsize = len(fmt.Sprintf("%v", response.Header))

	// measure response header time
	timestop = time.Now()
	respTime := (timestop.Sub(timestart))

	// defer close since we still want to read the body of the object
	defer response.Body.Close()

	// build a reader
	br := bufio.NewReader(response.Body)

	// now read the first byte
	_, err = br.ReadByte()
	if err != nil {
		if err != io.EOF {
			log.Printf("Worker %d: Could not read data: %s\n", w.id, err)
		}
		return
	}


	// measure our first byte time, this is normally 0ms however longer periods could be indicative of a problem
	timestop = time.Now()
	byteTime := (timestop.Sub(timestart))

	// ok, read the rest of the response
	n, err := br.Read(buffer)
	bsize = n
	for err != io.EOF {
		n, err = br.Read(buffer)
		bsize += n
	}

	// account for our 1st byte
	bsize += 1

	// did we fail to read everything?
	if err != nil && err != io.EOF {
		log.Printf("Worker %d: Error on data read, continuing with only %d bytes of %s read\n", w.id, bsize, response.Header.Get("Content-Length"))
	}

	// measure our overall time to proccess the entire transaction
	timestop = time.Now()
	totalTime := (timestop.Sub(timestart))


	// note that we're not shutting down our client or connections, so we can re-use them as needed

	// print out our values
	if ssl {
		log.Printf("Worker %d: %s -> DNS Lookup: %dms(cached: %t), Socket Connect: %dms, SSL Negotiation: %dms, Response Time: %dms, 1st Byte: %dms, Total Time: %dms, Size: %d\n", 
			w.id, work.page, iplookup / 1000000, cached, tcpConnect / 1000000, sslHandshake / 1000000, respTime / 1000000, byteTime / 1000000, totalTime / 1000000, bsize+hsize)
	} else {
		log.Printf("Worker %d: %s -> DNS Lookup: %dms(cached: %t), Socket Connect: %dms, Response Time: %dms, 1st Byte: %dms, Total Time: %dms, Size: %d\n", 
			w.id, work.page, iplookup / 1000000, cached, tcpConnect / 1000000, respTime / 1000000, byteTime / 1000000, totalTime / 1000000, bsize+hsize)
	}

	return

}

func NewWorker(id int, last string, client *httputil.ClientConn, in, out chan *Work, control chan int) *Worker {
	
	return &Worker{id: id, last: last, client: client, in: in, out: out, control: control}
}

func (w *Worker) StartWorker() {

	for work := range w.in {
		w.safeTestObject(work)
		w.out <- work
	}

	w.control <- w.id
}
*/

func CacheLookup (hostname string) (ipaddr string, elapsed time.Duration, cached bool, err error) {

	// assume we do not have it in cache
	cached = false

	// start a timer
	tstart := time.Now()

	// do we already have it mapped?
	if cache4[hostname] != "" {
		ipaddr = cache4[hostname]
		elapsed = 0 * time.Nanosecond
		err = nil
		cached = true
		return
	}

	// do a lookup
	addr, err := net.LookupHost(hostname)
	if err != nil {
		ipaddr = ""
		elapsed = 0 * time.Nanosecond
		return
	}

	// stop timer and get results
	tstop := time.Now()
	elapsed = tstop.Sub(tstart)

	for _, x := range addr {
		if len(x) > 16 {
			cache6[hostname] = x
		} else {
			cache4[hostname] = x
			ipaddr = x
		}
	}

	return
}

func ParseScript(text []byte) (newpage string) {

	log.Printf("Got Text: %s", string(text))
	return
}
