package proxy

import (
  "log"
  "crypto/tls"
  "io"
  "net"
)

// Proxy - Manages a Proxy connection, piping data between local and remote.
type Proxy struct {
  sentBytes     uint64
  receivedBytes uint64
  laddr, raddr  *net.TCPAddr
  lconn, rconn  io.ReadWriteCloser
  erred         bool
  errsig        chan bool
  tlsUnwrapp    bool
  tlsAddress    string

  Matcher  func([]byte)
  Replacer func([]byte) []byte

  // Settings
  Nagles    bool
  OutputHex bool
}

// New - Create a new Proxy instance. Takes over local connection passed in,
// and closes it when finished.
func New(lconn *net.TCPConn, laddr, raddr *net.TCPAddr) *Proxy {
  return &Proxy{
    lconn:  lconn,
    laddr:  laddr,
    raddr:  raddr,
    erred:  false,
    errsig: make(chan bool),
  }
}

// NewTLSUnwrapped - Create a new Proxy instance with a remote TLS server for
// which we want to unwrap the TLS to be able to connect without encryption
// locally
func NewTLSUnwrapped(lconn *net.TCPConn, laddr, raddr *net.TCPAddr, addr string) *Proxy {
  p := New(lconn, laddr, raddr)
  p.tlsUnwrapp = true
  p.tlsAddress = addr
  return p
}

type setNoDelayer interface {
  SetNoDelay(bool) error
}

// Start - open connection to remote and start proxying data.
func (p *Proxy) Start() {
  defer p.lconn.Close()

  var err error
  //connect to remote
  if p.tlsUnwrapp {
    p.rconn, err = tls.Dial("tcp", p.tlsAddress, nil)
  } else {
    p.rconn, err = net.DialTCP("tcp", nil, p.raddr)
  }
  if err != nil {
    log.Printf("warn: Remote connection failed: %s", err.Error())
    return
  }
  defer p.rconn.Close()

  //nagles?
  if p.Nagles {
    if conn, ok := p.lconn.(setNoDelayer); ok {
      conn.SetNoDelay(true)
    }
    if conn, ok := p.rconn.(setNoDelayer); ok {
      conn.SetNoDelay(true)
    }
  }

  //display both ends
  log.Printf("info: Opened %s >>> %s", p.laddr.String(), p.raddr.String())

  //bidirectional copy
  go p.pipe(p.lconn, p.rconn)
  go p.pipe(p.rconn, p.lconn)

  //wait for close...
  <-p.errsig
  log.Printf("info: Closed (%d bytes sent, %d bytes recieved)", p.sentBytes, p.receivedBytes)
}

func (p *Proxy) err(s string, err error) {
  if p.erred {
    return
  }
  if err != io.EOF {
    log.Printf(s, err.Error())
  }
  p.errsig <- true
  p.erred = true
}

func (p *Proxy) pipe(src, dst io.ReadWriter) {
  islocal := src == p.lconn

  var dataDirection string
  if islocal {
    dataDirection = "debug: >>> %d bytes sent"
  } else {
    dataDirection = "debug: <<< %d bytes recieved"
  }

  var byteFormat string
  if p.OutputHex {
    byteFormat = "trace: %x"
  } else {
    byteFormat = "trace: %s"
  }

  //directional copy (64k buffer)
  buff := make([]byte, 0xffff)
  for {
    n, err := src.Read(buff)
    if err != nil {
      p.err("warn: Read failed '%s'", err)
      return
    }
    b := buff[:n]

    //execute match
    if p.Matcher != nil {
      p.Matcher(b)
    }

    //execute replace
    if p.Replacer != nil {
      b = p.Replacer(b)
    }

    //show output
    log.Printf(dataDirection, n)
    log.Printf(byteFormat, b)

    //write out result
    n, err = dst.Write(b)
    if err != nil {
      p.err("warn: Write failed '%s'", err)
      return
    }
    if islocal {
      p.sentBytes += uint64(n)
    } else {
      p.receivedBytes += uint64(n)
    }
  }
}
