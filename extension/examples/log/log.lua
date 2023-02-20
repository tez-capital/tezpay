--[[
This is an example of a eli (Lua) extension.
It logs all requests and responses to a file.
Path to log file is specified in tezpay configuration.

extensions: {
	log: {
		path: "extension/examples/log.lua",
		configuration: {
			LOG_FILE: "extension/examples/log.txt"
		}
	}
}
]]

local hjson = require("hjson")

local METHOD_NOT_FOUND = { code = -32601, message = "Method not found" }
local def = { configuration = {} }

local function append_to_file(data)
	if def.configuration.LOG_FILE ~= nil then
		local LOG_FILE = def.configuration.LOG_FILE
		fs.write_file(LOG_FILE, data, { append = true })
	end
end

local function stringify(value)
	return hjson.stringify_to_json(value, { indent = false })
end

local function write_error(id, error)
	local response = stringify({ jsonrpc = "2.0", id = id, error = METHOD_NOT_FOUND })
	io.write(response .. "\n")
	io.output():flush()
end

local function write_response(id, result)
	local response = stringify({ jsonrpc = "2.0", id = id, result = result })
	io.write(response .. "\n")
	io.output():flush()
end

while true do
	local line = io.read()
	local request = hjson.parse(line)
	local id = request.id
	local method = request.method
	if method == "test-request" then
		local data = request.params.data
		data.message = "Hello from Lua!"
		write_response(id, data)
	elseif method == "close" then
		os.exit(0)
	elseif method == "initialize" then
		def = util.merge_tables(def, request.params.definition, true)
		write_response(id, {
			success = true,
		})
		append_to_file("definition:" .. "\n")
		append_to_file(stringify(def) .. "\n")
		append_to_file("owned by: " .. request.params.owner_id .. "\n")
	elseif method ~= nil and id ~= nil then
		-- extensions should return an error for unknown methods
		-- write_error(id, METHOD_NOT_FOUND)
		-- but log extension wants to log all data
		write_response(id, request.params.data)
	end

	append_to_file(line .. "\n")
end
