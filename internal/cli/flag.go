package cli

import "flag"

type Flags struct {
	ConfigPath string
	DevMode    bool
	DevOwner   string
	DevRepo    string
}

func ParseFlags() (*Flags, error) {
	flags := &Flags{}
	flag.StringVar(&flags.ConfigPath, "config", "config.yaml", "path to config file")
	flag.BoolVar(&flags.DevMode, "dev", false, "enable dev mode (auto-starts gh webhook forward)")
	flag.StringVar(&flags.DevOwner, "owner", "", "owner for dev mode webhook forwarding")
	flag.StringVar(&flags.DevRepo, "repo", "", "repo for dev mode webhook forwarding")
	flag.Parse()
	return flags, nil
}
