package comic

import "github.com/spf13/viper"

func init() {
	viper.SetDefault("comic.verify.concurrent", 10)
	viper.SetDefault("comic.verify.task_buffer_size", 100)
}
