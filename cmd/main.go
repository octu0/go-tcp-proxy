package main

import (
  "log"
  "fmt"
  "net"
  "os"
  "regexp"
  "strings"

  "github.com/comail/colog"
  "gopkg.in/urfave/cli.v1"

  "github.com/octu0/go-tcp-proxy"
)

var (
  Commands = make([]cli.Command, 0)
)
func AddCommand(cmd cli.Command){
  Commands = append(Commands, cmd)
}

func action(c *cli.Context) error {
  localAddr  := c.String("local")
  remoteAddr := c.String("remote")
  nagles     := c.Bool("no-nagles")
  unwrapTLS  := c.Bool("unwrap-tls")
  match      := c.String("match")
  replace    := c.String("replace")

  outputHex  := c.Bool("hex")
  debug      := c.Bool("debug")
  verbose    := c.Bool("verbose")

  log.Printf("info: starting up %s", proxy.UA)

  laddr, err := net.ResolveTCPAddr("tcp", localAddr)
  if err != nil {
    log.Printf("error: Failed to resolve local address(%s): %s", localAddr, err.Error())
    return err
  }
  raddr, err := net.ResolveTCPAddr("tcp", remoteAddr)
  if err != nil {
    log.Printf("error: Failed to resolve remote address(%s): %s", remoteAddr, err.Error())
    return err
  }
  listener, err := net.ListenTCP("tcp", laddr)
  if err != nil {
    log.Printf("error: Failed to open local port to listen(%s): %s", laddr, err.Error())
    return err
  }

  log.Printf("info: Proxying from %s to %s", localAddr, remoteAddr)

  matcher := createMatcher(match)
  replacer := createReplacer(replace)

  if verbose {
    debug = true
  }
  connid := uint64(0)
  for {
    conn, err := listener.AcceptTCP()
    if err != nil {
      log.Printf("warn: Failed to accept connection '%s'", err.Error())
      continue
    }
    connid++

    var p *proxy.Proxy
    if unwrapTLS {
      log.Printf("info: Unwrapping TLS enable")
      p = proxy.NewTLSUnwrapped(conn, laddr, raddr, remoteAddr)
    } else {
      p = proxy.New(conn, laddr, raddr)
    }

    p.Matcher = matcher
    p.Replacer = replacer

    p.Nagles = nagles
    p.OutputHex = outputHex
    p.Log = proxy.ColorLogger{
      Verbose:     debug,
      VeryVerbose: verbose,
      Prefix:      fmt.Sprintf("Connection #%03d ", connid),
      Color:       false,
    }

    go p.Start()

  }

  return nil
}

func createMatcher(match string) func([]byte) {
  if match == "" {
    return nil
  }
  re, err := regexp.Compile(match)
  if err != nil {
    log.Printf("warn: Invalid match regex(%s): %s", match, err.Error())
    return nil
  }

  log.Printf("info: matching %s", re.String())
  matchid := uint64(0)
  return func(input []byte) {
    ms := re.FindAll(input, -1)
    for _, m := range ms {
      matchid++
      log.Printf("info: Matched #%d: %s", matchid, string(m))
    }
  }
}

func createReplacer(replace string) func([]byte) []byte {
  if replace == "" {
    return nil
  }
  //split by / (TODO: allow slash escapes)
  parts := strings.Split(replace, "~")
  if len(parts) != 2 {
    log.Printf("warn: Invalid replace option:'%s'", replace)
    return nil
  }

  re, err := regexp.Compile(string(parts[0]))
  if err != nil {
    log.Printf("warn: Invalid replace regex(%s): %s", parts[0], err)
    return nil
  }

  repl := []byte(parts[1])

  log.Printf("info: replacing %s with %s", re.String(), repl)
  return func(input []byte) []byte {
    return re.ReplaceAll(input, repl)
  }
}

func main(){
  colog.SetDefaultLevel(colog.LDebug)
  colog.SetMinLevel(colog.LInfo)

  colog.SetFormatter(&colog.StdFormatter{
    Flag: log.Ldate | log.Ltime | log.Lshortfile,
  })
  colog.Register()

  app         := cli.NewApp()
  app.Version  = proxy.Version
  app.Name     = proxy.AppName
  app.Author   = ""
  app.Email    = ""
  app.Usage    = ""
  app.Action   = action
  app.Commands = Commands
  app.Flags    = []cli.Flag{
    cli.StringFlag{
      Name: "l, local",
      Usage: "local address",
      Value: ":9999",
      EnvVar: "TCPPROXY_LOCALADDR",
    },
    cli.StringFlag{
      Name: "r, remote",
      Usage: "remote(destination) address",
      Value: "127.0.0.1:8000",
      EnvVar: "TCPPROXY_REMOTEADDR",
    },
    cli.BoolFlag{
      Name: "n, no-nagles",
      Usage: "disable nagles algorithm",
      EnvVar: "TCPPROXY_DISABLE_NAGLES",
    },
    cli.BoolFlag{
      Name: "unwrap-tls",
      Usage: "remote connection with TLS exposed unencrypted locally",
      EnvVar: "TCPPROXY_UNWRAP_TLS",
    },
    cli.StringFlag{
      Name: "match",
      Usage: "match regex(in the form `regex`)",
      Value: "",
      EnvVar: "TCPPROXY_MATCH",
    },
    cli.StringFlag{
      Name: "replace",
      Usage: "replace regex(in the form `regex-replacer`)",
      Value: "",
      EnvVar: "TCPPROXY_REPLACE",
    },
    cli.BoolFlag{
      Name: "d, debug",
      Usage: "display server actions",
      EnvVar: "TCPPROXY_DEBUG",
    },
    cli.BoolFlag{
      Name: "V, verbose",
      Usage: "display server actions and all tcp data",
      EnvVar: "TCPPROXY_VERBOSE",
    },
    cli.BoolFlag{
      Name: "hex",
      Usage: "output binary data hexdecimal",
      EnvVar: "TCPPROXY_OUTPUT_HEX",
    },
  }
  if err := app.Run(os.Args); err != nil {
    log.Printf("error: %s", err.Error())
    cli.OsExiter(1)
  }
}
