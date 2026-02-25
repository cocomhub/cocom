/*
Copyright © 2023 suixibing <suixibing@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package config

import (
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("cocom.storage.path", "/data/cocom/data/gallery")
	viper.SetDefault("cocom.archive.path", "/data/cocom/data/archive")
	viper.SetDefault("cocom.archive.password", "")
	viper.SetDefault("cocom.archive.cmd", "7z")
	viper.SetDefault("cocom.archive.algorithm", "double")
}

func GetSaveRoot() string {
	return viper.GetString("cocom.storage.path")
}

func GetArchiveRoot() string {
	return viper.GetString("cocom.archive.path")
}

func GetArchivePassword() string {
	return viper.GetString("cocom.archive.password")
}

func GetArchiveCmd() string {
	return viper.GetString("cocom.archive.cmd")
}

func GetArchiveAlgorithm() string {
	return viper.GetString("cocom.archive.algorithm")
}
