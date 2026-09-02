package flags

import "flag"

type Options struct {
	File    string
	DB      bool
	Version bool
	Seed    bool
}

var FlagOptions = new(Options)

func Parse() {
	flag.StringVar(&FlagOptions.File, "f", "settings.yaml", "配置文件")
	flag.BoolVar(&FlagOptions.DB, "db", false, "数据库迁移")
	flag.BoolVar(&FlagOptions.Version, "v", false, "版本")
	flag.BoolVar(&FlagOptions.Seed, "seed", false, "填充种子数据")
	flag.Parse()
}
