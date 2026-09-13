package command

import (
	"fmt"
	"strings"

	role "gtps/VallenSource/VallenRole"
)

// Permission levels
const (
	LevelPlayer = role.LevelPlayer          // 0
	LevelMod    = role.LevelEliteGuardian   // 5
	LevelAdmin  = role.LevelGrandArchitect  // 7
	LevelDev    = role.LevelShadowSovereign // 9
	LevelOwner  = role.LevelMonarch         // 999
)

type HandlerFunc func(c *Ctx)

type Command struct {
	Name     string
	Aliases  []string
	Usage    string
	Desc     string
	MinArgs  int
	MinLevel int
	Handler  HandlerFunc
}

// registry menyimpan semua command yang terdaftar
var registry = map[string]*Command{}

// Add mendaftarkan command baru dengan chaining (gaya code malas / abang-abangan)
func Add(name string, minLevel int) *Command {
	cmd := &Command{
		Name:     strings.ToLower(name),
		MinLevel: minLevel,
	}
	registry[cmd.Name] = cmd
	return cmd
}

func (cmd *Command) Alias(aliases ...string) *Command {
	cmd.Aliases = append(cmd.Aliases, aliases...)
	for _, a := range aliases {
		registry[strings.ToLower(a)] = cmd
	}
	return cmd
}

func (cmd *Command) SetUsage(usage string) *Command {
	cmd.Usage = usage
	return cmd
}

func (cmd *Command) SetDesc(desc string) *Command {
	cmd.Desc = desc
	return cmd
}

func (cmd *Command) SetMinArgs(n int) *Command {
	cmd.MinArgs = n
	return cmd
}

func (cmd *Command) Exec(fn HandlerFunc) *Command {
	cmd.Handler = fn
	return cmd
}

// Get mencari command berdasarkan nama atau alias
func Get(name string) (*Command, bool) {
	cmd, ok := registry[strings.ToLower(name)]
	return cmd, ok
}

// GetAll mengembalikan daftar semua command unik
func GetAll() []*Command {
	seen := make(map[string]bool)
	var list []*Command
	for _, cmd := range registry {
		if !seen[cmd.Name] {
			seen[cmd.Name] = true
			list = append(list, cmd)
		}
	}
	return list
}

// DispatchResult hasil eksekusi command
type DispatchResult int

const (
	DispatchOK DispatchResult = iota
	DispatchNotFound
	DispatchPermissionDenied
	DispatchMissingArgs
)

// Dispatch memproses raw text command dan mengeksekusi handlernya
func Dispatch(c *Ctx) DispatchResult {
	parts := strings.Fields(c.RawText)
	if len(parts) == 0 {
		return DispatchNotFound
	}

	cmdName := strings.TrimPrefix(strings.ToLower(parts[0]), "/")
	c.Args = parts[1:]

	cmd, ok := Get(cmdName)
	if !ok {
		return DispatchNotFound
	}

	if c.RoleLevel < cmd.MinLevel {
		return DispatchPermissionDenied
	}

	if len(c.Args) < cmd.MinArgs {
		return DispatchMissingArgs
	}

	if cmd.Handler != nil {
		cmd.Handler(c)
	}
	return DispatchOK
}

// Format helpers
func FormatUsage(usage string) string {
	return fmt.Sprintf("`4Usage: `5%s``", usage)
}

func FormatError(msg string) string {
	return fmt.Sprintf("`4%s``", msg)
}

func FormatSuccess(msg string) string {
	return fmt.Sprintf("`2%s``", msg)
}

func FormatInfo(msg string) string {
	return fmt.Sprintf("`9%s``", msg)
}
