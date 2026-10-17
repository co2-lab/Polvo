// @anchors
//   code: CLMAN
//   ref: CLIXX

// Command polvo is the desktop, the node and the bridge of the local agent, in one executable.
package main

import "os"

func main() {
	os.Exit(Run(os.Args[1:], Stdio{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}))
}
