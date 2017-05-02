package ssh

import (
	"fmt"
	"github.com/JeanSebTr/SshBrain/domain"
	"log"
	"strings"
)

type CmdContext struct {
	domain.Channel
	Log     *log.Logger
	Manager domain.NodeManager
	Pty     *domain.PtyRequest
}

type Cmd struct {
	desc string
	cb   func(CmdContext, Arguments) int
}

type Cmds map[string]Cmd

var commands Cmds

func (c Cmds) Exec(ctx CmdContext, cmd string) int {
	args := strings.Split(cmd, " ")
	if cmd, exists := commands[args[0]]; exists {
		return cmd.cb(ctx, args[1:])
	}
	// TODO: fix out of order output
	fmt.Fprintf(ctx.Stderr(), "%s: Command not found\r\n", cmd)
	return 127
}

func init() {
	commands = map[string]Cmd{
		"help": Cmd{"This help text", func(ctx CmdContext, _ Arguments) int {
			for name, cmd := range commands {
				fmt.Fprintf(ctx, "%s\t%s\r\n", name, cmd.desc)
			}
			return 0
		}},
		"devices": Cmd{"List connected devices", func(ctx CmdContext, _ Arguments) int {
			fmt.Fprintln(ctx, "Id\tAddr\tServices\r")
			for _, node := range ctx.Manager.GetAll() {
				fmt.Fprintf(ctx, "%s\t%s\r\n", node.Id(), node.Address())
			}
			return 0
		}},
		"connect": Cmd{"Establish a SSH connection to a device", func(ctx CmdContext, args Arguments) int {
			log.Printf("Trying to connect to %v\n", args)
			if len(args) < 1 {
				fmt.Fprintln(ctx.Stderr(), "Missing client ID\r")
				return 126
			}

			target := args[0]

			if node, err := ctx.Manager.GetById(target); err != nil || node == nil {
				if err != nil {
					ctx.Log.Printf("Error finding node id %s: %s\n", target, err)
				}
				fmt.Fprintf(ctx.Stderr(), "Node id %s not found\r\n", target)
			} else if session, err := node.NewSession(ctx.Channel, ctx.Pty); err != nil {
				ctx.Log.Printf("Error creating session on node id %s: %s\n", target, err)
				fmt.Fprintf(ctx.Stderr(), "Error connecting to %s\r\n", target)
			} else if err = session.Shell(); err != nil {
				ctx.Log.Printf("Error opening shell on node id %s: %s\n", target, err)
				fmt.Fprintf(ctx.Stderr(), "Error connecting to %s\r\n", target)
			} else {
				return 0
			}
			return 126
		}},
		"scp": Cmd{"Copy data to remote nodes", func(ctx CmdContext, args Arguments) int {
			path, err := args.Single(func(str string) bool {
				return len(str) > 0 && str[0] == '/'
			})
			if err != nil {
				fmt.Fprintln(ctx.Stderr(), "Destination path not found in command")
				return 126
			}

			target := path[1:]
			nodePath := "/"
			if i := strings.IndexByte(target, '/'); i != -1 {
				target = path[1 : i+1]
				nodePath = path[i+1:]
			}

			newArgs := make(Arguments, len(args)+1)
			newArgs[0] = "scp"
			for i, arg := range args {
				if arg == path {
					newArgs[i+1] = nodePath
				} else {
					newArgs[i+1] = arg
				}
			}

			cmd := newArgs.String()

			if node, err := ctx.Manager.GetById(target); err != nil || node == nil {
				if err != nil {
					ctx.Log.Printf("Error finding node id %s: %s\n", target, err)
				}
				fmt.Fprintf(ctx.Stderr(), "Node id %s not found\r\n", target)
			} else if session, err := node.NewSession(ctx.Channel, ctx.Pty); err != nil {
				ctx.Log.Printf("Error creating session on node id %s: %s\n", target, err)
				fmt.Fprintf(ctx.Stderr(), "Error connecting to %s\r\n", target)
			} else if exitCode, err := session.Exec(cmd); err != nil {
				ctx.Log.Printf("Error running command `%s` on node id %s: %s\n", cmd, target, err)
				fmt.Fprintf(ctx.Stderr(), "Error running command `%s` on %s\r\n", cmd, target)
			} else {
				return exitCode
			}

			return 126
		}},
	}
}
