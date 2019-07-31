package proxy

type MatcherFunc  func([]byte)
type ReplacerFunc func([]byte) []byte

type Options struct {
  tlsUnwrap    bool
  tlsAddress   string
  nagles       bool
  matcher      MatcherFunc
  replacer     ReplacerFunc
  outputHex    bool
  debugMode    bool
  verboseMode  bool
}

type OptionsFunc func(opts *Options)

func TLSUnwrap(tlsUnwrap bool) OptionsFunc {
  return func(opts *Options) {
    opts.tlsUnwrap = tlsUnwrap
  }
}
func TLSAddress(tlsAddress string) OptionsFunc {
  return func(opts *Options) {
    opts.tlsAddress = tlsAddress
  }
}
func Nagles(nagles bool) OptionsFunc {
  return func(opts *Options) {
    opts.nagles = nagles
  }
}
func Matcher(matcher MatcherFunc) OptionsFunc {
  return func(opts *Options) {
    opts.matcher = matcher
  }
}
func Replacer(replacer ReplacerFunc) OptionsFunc {
  return func(opts *Options) {
    opts.replacer = replacer
  }
}
func OutputHex(outputHex bool) OptionsFunc {
  return func(opts *Options) {
    opts.outputHex = outputHex
  }
}
func DebugMode(debugMode bool) OptionsFunc {
  return func(opts *Options) {
    opts.debugMode = debugMode
  }
}
func VerboseMode(verboseMode bool) OptionsFunc {
  return func(opts *Options) {
    opts.verboseMode = verboseMode
  }
}
