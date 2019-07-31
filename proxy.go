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
  lconn         io.ReadWriteCloser
  rconn         io.ReadWriteCloser
  laddr         *net.TCPAddr
  raddr         *net.TCPAddr
  hasError      bool
  errChan       chan bool
  opts          *Options
}

func New(lconn io.ReadWriteCloser, laddr *net.TCPAddr, raddr *net.TCPAddr, optsFunc ...OptionsFunc) *Proxy {
  opts := new(Options)
  for _, f := range optsFunc {
    f(opts)
  }

  p := new(Proxy)
  p.lconn    = lconn
  p.laddr    = laddr
  p.raddr    = raddr
  p.hasError = false
  p.errChan  = make(chan bool)
  p.opts     = opts
  return p
}

type setNoDelayer interface {
  SetNoDelay(bool) error
}

// Start - open connection to remote and start proxying data.
func (p *Proxy) Start() {
  defer p.lconn.Close()

  var err error
  if p.opts.tlsUnwrap {
    p.rconn, err = tls.Dial("tcp", p.opts.tlsAddress, nil)
  } else {
    p.rconn, err = net.DialTCP("tcp", nil, p.raddr)
  }
  if err != nil {
    log.Printf("warn: Remote connection failed: %s", err.Error())
    return
  }
  defer p.rconn.Close()

  //nagles?
  if p.opts.nagles {
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
  <-p.errChan
  log.Printf("info: Closed (%d bytes sent, %d bytes recieved)", p.sentBytes, p.receivedBytes)
}

func (p *Proxy) err(s string, err error) {
  if p.hasError {
    return
  }
  if err != io.EOF {
    log.Printf(s, err.Error())
  }
  p.errChan <- true
  p.hasError = true
}

func (p *Proxy) pipe(src, dst io.ReadWriter) {
  islocal := src == p.lconn

  buff := make([]byte, 0xffff)
  for {
    n, err := src.Read(buff)
    if err != nil {
      p.err("warn: Read failed '%s'", err)
      return
    }
    b := buff[:n]

    //execute match
    if p.opts.matcher != nil {
      p.opts.matcher(b)
    }

    //execute replace
    if p.opts.replacer != nil {
      b = p.opts.replacer(b)
    }

    if p.opts.debugMode {
      if islocal {
        log.Printf("debug: >>> %d bytes sent", n)
      } else {
        log.Printf("debug: <<< %d bytes recieved", n)
      }
    }
    if p.opts.verboseMode {
      if p.opts.outputHex {
        log.Printf("trace: %x", b)
      } else {
        log.Printf("trace: %s", b)
      }
    }

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
