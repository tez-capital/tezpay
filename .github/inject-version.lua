local version = ...

assert(version, "no version provided")

local VERSION_FILE = "constants/version.go"

local file = fs.read_file(VERSION_FILE)

--[[
package constants

const (
	VERSION  = "dev"
	CODENAME = "tezpay"
)
]]

file = file:gsub('VERSION%s*=%s*"dev"', 'VERSION = "' .. version .. '"')
print(file)
fs.write_file(VERSION_FILE, file)