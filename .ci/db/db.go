package db

import "gorm.io/gorm"

var Dao *gorm.DB

func GetDBPath() string { return "./data/stock.db" }
