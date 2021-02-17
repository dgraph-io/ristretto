package z

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// SuperFlagHelp makes it really easy to generate command line `--help` output for a SuperFlag. For
// example:
//
//	const flagDefaults = `enabled=true; path=some/path;`
//
//  var help string = z.NewSuperFlagHelp(flagDefaults).
//		Flag("enabled", "Turns on <something>.").
//		Flag("path", "The path to <something>.").
//		Flag("another", "Not present in defaults, but still included.").
//		String()
//
// The `help` string would then contain:
//
//	enabled=true; Turns on <something>.
//	path=some/path; The path to <something>.
//	another=; Not present in defaults, but still included.
//
// All flags are sorted alphabetically for consistent `--help` output. Flags with default values are
// placed at the top, and everything else goes under.
type SuperFlagHelp struct {
	defaults *SuperFlag
	flags    map[string]string
}

func NewSuperFlagHelp(defaults string) *SuperFlagHelp {
	return &SuperFlagHelp{
		defaults: NewSuperFlag(defaults),
		flags:    make(map[string]string, 0),
	}
}

func (h *SuperFlagHelp) Flag(name, description string) *SuperFlagHelp {
	h.flags[name] = description
	return h
}

func (h *SuperFlagHelp) String() string {
	defaultLines := make([]string, 0)
	otherLines := make([]string, 0)
	for name, help := range h.flags {
		val, found := h.defaults.m[name]
		line := fmt.Sprintf("%s=%s; %s\n", name, val, help)
		if found {
			defaultLines = append(defaultLines, line)
		} else {
			otherLines = append(otherLines, line)
		}
	}
	sort.Strings(defaultLines)
	sort.Strings(otherLines)
	return strings.Join(defaultLines, "") +
		//		"---\n" +
		strings.Join(otherLines, "")
}

func parseFlag(flag string) map[string]string {
	kvm := make(map[string]string)
	for _, kv := range strings.Split(flag, ";") {
		if strings.TrimSpace(kv) == "" {
			continue
		}
		splits := strings.SplitN(kv, "=", 2)
		k := strings.TrimSpace(splits[0])
		k = strings.ToLower(k)
		k = strings.ReplaceAll(k, "_", "-")
		kvm[k] = strings.TrimSpace(splits[1])
	}
	return kvm
}

type SuperFlag struct {
	m map[string]string
}

func NewSuperFlag(flag string) *SuperFlag {
	return &SuperFlag{
		m: parseFlag(flag),
	}
}

func (sf *SuperFlag) String() string {
	if sf == nil {
		return ""
	}
	var kvs []string
	for k, v := range sf.m {
		kvs = append(kvs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(kvs, "; ")
}

func (sf *SuperFlag) MergeAndCheckDefault(flag string) *SuperFlag {
	if sf == nil {
		sf = &SuperFlag{
			m: parseFlag(flag),
		}
		return sf
	}
	numKeys := len(sf.m)
	src := parseFlag(flag)
	for k := range src {
		if _, ok := sf.m[k]; ok {
			numKeys--
		}
	}
	if numKeys != 0 {
		msg := fmt.Sprintf("Found invalid options in %s. Valid options: %v", sf, flag)
		panic(msg)
	}
	for k, v := range src {
		if _, ok := sf.m[k]; !ok {
			sf.m[k] = v
		}
	}
	return sf
}

func (sf *SuperFlag) Has(opt string) bool {
	val := sf.GetString(opt)
	return val != ""
}

func (sf *SuperFlag) GetDuration(opt string) time.Duration {
	val := sf.GetString(opt)
	if val == "" {
		return time.Duration(0)
	}
	if strings.Contains(val, "d") {
		val = strings.Replace(val, "d", "", 1)
		days, err := strconv.ParseUint(val, 0, 64)
		if err != nil {
			return time.Duration(0)
		}
		return time.Hour * 24 * time.Duration(days)
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return time.Duration(0)
	}
	return d
}

func (sf *SuperFlag) GetBool(opt string) bool {
	val := sf.GetString(opt)
	if val == "" {
		return false
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		err = errors.Wrapf(err,
			"Unable to parse %s as bool for key: %s. Options: %s\n",
			val, opt, sf)
		log.Fatalf("%+v", err)
	}
	return b
}

func (sf *SuperFlag) GetFloat64(opt string) float64 {
	val := sf.GetString(opt)
	if val == "" {
		return 0
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		err = errors.Wrapf(err,
			"Unable to parse %s as float64 for key: %s. Options: %s\n",
			val, opt, sf)
		log.Fatalf("%+v", err)
	}
	return f
}

func (sf *SuperFlag) GetInt64(opt string) int64 {
	val := sf.GetString(opt)
	if val == "" {
		return 0
	}
	i, err := strconv.ParseInt(val, 0, 64)
	if err != nil {
		err = errors.Wrapf(err,
			"Unable to parse %s as int64 for key: %s. Options: %s\n",
			val, opt, sf)
		log.Fatalf("%+v", err)
	}
	return i
}

func (sf *SuperFlag) GetUint64(opt string) uint64 {
	val := sf.GetString(opt)
	if val == "" {
		return 0
	}
	u, err := strconv.ParseUint(val, 0, 64)
	if err != nil {
		err = errors.Wrapf(err,
			"Unable to parse %s as uint64 for key: %s. Options: %s\n",
			val, opt, sf)
		log.Fatalf("%+v", err)
	}
	return u
}

func (sf *SuperFlag) GetUint32(opt string) uint32 {
	val := sf.GetString(opt)
	if val == "" {
		return 0
	}
	u, err := strconv.ParseUint(val, 0, 32)
	if err != nil {
		err = errors.Wrapf(err,
			"Unable to parse %s as uint32 for key: %s. Options: %s\n",
			val, opt, sf)
		log.Fatalf("%+v", err)
	}
	return uint32(u)
}

func (sf *SuperFlag) GetString(opt string) string {
	if sf == nil {
		return ""
	}
	return sf.m[opt]
}
