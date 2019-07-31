package proxy

import (
  "log"
  "fmt"
  "time"
  "io"
  "net"
  "crypto/tls"
)

const(
  KilloByte uint64 = 1024
  MegaByte  uint64 = KilloByte * 1024
  GigaByte  uint64 = MegaByte  * 1024
)

type Proxy struct {
  startAt       time.Time
  sendBytes     uint64
  receiveBytes  uint64
  laddr         *net.TCPAddr
  raddr         *net.TCPAddr
  closed        bool
  closeChan     chan bool
  monCloseChan  chan bool
  opts          *Options
}

func New(laddr, raddr *net.TCPAddr, optsFunc ...OptionsFunc) *Proxy {
  opts := new(Options)
  for _, f := range optsFunc {
    f(opts)
  }

  p             := new(Proxy)
  p.startAt      = time.Now()
  p.laddr        = laddr
  p.raddr        = raddr
  p.closed       = false
  p.closeChan    = make(chan bool)
  p.monCloseChan = make(chan bool)
  p.opts         = opts
  return p
}

type setNoDelayer interface {
  SetNoDelay(bool) error
}

func (p *Proxy) Start(lconn net.Conn) {
  defer lconn.Close()

  var err error
  var rconn net.Conn
  if p.opts.tlsUnwrap {
    rconn, err = tls.Dial("tcp", p.opts.tlsAddress, nil)
  } else {
    rconn, err = net.DialTCP("tcp", nil, p.raddr)
  }
  if err != nil {
    log.Printf("warn: Remote connection failed: %s", err.Error())
    return
  }
  defer rconn.Close()

  if p.opts.nagles {
    if conn, ok := lconn.(setNoDelayer); ok {
      conn.SetNoDelay(true)
    }
    if conn, ok := rconn.(setNoDelayer); ok {
      conn.SetNoDelay(true)
    }
  }

  clientAddr := lconn.RemoteAddr().String()
  remoteAddr := rconn.RemoteAddr().String()

  log.Printf("info: start proxy %s to %s", clientAddr, remoteAddr)

  go p.pipe(lconn, rconn, clientAddr, remoteAddr, true)
  go p.pipe(rconn, lconn, clientAddr, remoteAddr, false)
  go p.monitor(clientAddr, remoteAddr)

  <-p.closeChan

  log.Printf("info: close proxy %s to %s (%s)", clientAddr, remoteAddr, p.stat())
}

func (p *Proxy) monitor(srcAddr, dstAddr string) {
  ticker  := time.NewTicker(30 * time.Second)
  running := true
  log.Printf("debug: monitor starting")
  for running {
    select {
    case <-ticker.C:
      log.Printf("debug: stat %s to %s (%s)", srcAddr, dstAddr, p.stat())
    case <-p.monCloseChan:
      running = false
    }
  }
  ticker.Stop()
}

func (p *Proxy) stat() string {
  txByte  := p.formatByte(p.sendBytes)
  rxByte  := p.formatByte(p.receiveBytes)
  elapsed := time.Since(p.startAt).String()
  return fmt.Sprintf("dur: %s tx: %s rx: %s", elapsed, txByte, rxByte)
}

func (p *Proxy) formatByte(bytes uint64) string {
    if GigaByte < bytes {
    return fmt.Sprintf("%3.2f GB", float64(bytes) / float64(GigaByte))
  }
  if MegaByte < bytes {
    return fmt.Sprintf("%3.2f MB", float64(bytes) / float64(MegaByte))
  }
  return fmt.Sprintf("%3.2f KB", float64(bytes) / float64(KilloByte))
}

func (p *Proxy) handleError(err error, s string) {
  if p.closed {
    return
  }
  if err != io.EOF {
    log.Printf(s)
  }
  p.closeChan <- true
  p.monCloseChan <- true
  p.closed = true
}

func (p *Proxy) pipe(src, dst io.ReadWriter, srcAddr, dstAddr string, islocal bool) {
  buff := make([]byte, 0xffff)
  for {
    n, err := src.Read(buff)
    if err != nil {
      p.handleError(err, fmt.Sprintf("warn: %s failed to read '%s'", srcAddr, err.Error()))
      return
    }
    b := buff[:n]

    if p.opts.matcher != nil {
      p.opts.matcher(b)
    }

    if p.opts.replacer != nil {
      b = p.opts.replacer(b)
    }

    if p.opts.debugMode {
      if islocal {
        log.Printf("debug: tx: %s to %s (%s)", srcAddr, dstAddr, p.formatByte(uint64(n)))
      } else {
        log.Printf("debug: rx: %s to %s (%s)", dstAddr, srcAddr, p.formatByte(uint64(n)))
      }
    }
    if p.opts.verboseMode {
      if p.opts.outputHex {
        log.Printf("trace: data=%x", b)
      } else {
        log.Printf("trace: data=%s", b)
      }
    }

    n, err = dst.Write(b)
    if err != nil {
      p.handleError(err, fmt.Sprintf("warn: %s failed to write '%s'", dstAddr, err.Error()))
      return
    }

    if islocal {
      p.sendBytes += uint64(n)
    } else {
      p.receiveBytes += uint64(n)
    }
  }
}
