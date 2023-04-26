--[[
Sample extension injecting data to notifications

extensions: [
	{
		command: eli
		args: ["extensions/enrich.lua"]
		configuration: {
			log_file = "enrich-notifications.log"
		}
	}
]
]]
local def                            = {
	configuration = {
		log_file = "enrich-notifications.log"
	}
}

local hjson                          = require("hjson")
local bigint                         = require("bigint")

-- calls
local CALL_PREFIX                    = "tp."
local CLOSE_CALL                     = "close"
local INIT_CALL                      = "init"
local HEALTHCHECK_CALL               = "healthcheck"

-- hooks
local TEST_REQUEST_HOOK              = "test-request"
local EXTENSION_HOOK_COLLECT_ADDITIONAL_NOTIFICATION_DATA = "collect_additional_notification_data"


local METHOD_NOT_FOUND = { code = -32601, message = "Method not found" }
local INVALID_REQUEST = { code = -32600, message = "Invalid request" }

local function new_server_error(data)
	local SERVER_ERROR = { code = -32000, message = "Server error", data = data }
	return SERVER_ERROR
end

local function validate_configuration()
	if not table.includes(def.hooks, EXTENSION_HOOK_COLLECT_ADDITIONAL_NOTIFICATION_DATA .. ":rw") then
		return new_server_error("hook " .. EXTENSION_HOOK_COLLECT_ADDITIONAL_NOTIFICATION_DATA .. " has to be enabled in rw mode")
	end
end

local function validate_request(request)
	if type(request) ~= "table" then
		return INVALID_REQUEST
	end
	if type(request.method) ~= "string" then
		return INVALID_REQUEST
	end
	if request.method:sub(1, #CALL_PREFIX) ~= CALL_PREFIX then
		return new_server_error("Method must start with " .. CALL_PREFIX)
	end
end

local function log(message)
	if type(def.configuration.log_file) == "string" and #def.configuration.log_file > 0 then
		fs.write_file(def.configuration.log_file, tostring(message) .. "\n", { append = true })
	end
end

local function stringify(value)
	return hjson.stringify_to_json(value, { indent = false })
end

local function write_error(id, error)
	local response = stringify({ jsonrpc = "2.0", id = id, error = error })
	log("ERROR: " .. response)
	io.write(response .. "\n")
	io.output():flush()
end

local function write_response(id, result)
	local response = stringify({ jsonrpc = "2.0", id = id, result = result })
	log("response: " .. response)
	io.write(response .. "\n")
	io.output():flush()
end

local handlers = {
	[INIT_CALL] = function(request)
		local id = request.id
		def = util.merge_tables(def, request.params.definition, true)
		def.bakerPkh = request.params.baker_pkh
		def.payoutPkh = request.params.payout_pkh
		local error = validate_configuration()
		if error ~= nil then
			write_error(id, error)
			write_response(id, {
				success = false,
				error = error,
			})
			os.exit(1)
			return
		end

		write_response(id, {
			success = true,
		})
	end,
	[CLOSE_CALL] = function()
		os.exit(0)
	end,
	[HEALTHCHECK_CALL] = function()
	end,
	[TEST_REQUEST_HOOK] = function(request)
		local data = request.params.data
		data.message = "Hello from Lua!"
		write_response(request.id, data)
	end,
	[EXTENSION_HOOK_COLLECT_ADDITIONAL_NOTIFICATION_DATA] = function(request)
		local id = request.id
		local version = request.params.version
		if version ~= "0.1" then
			write_error(id, new_server_error("Unsupported version: " .. version))
			return
		end

		local summary = request.params.data
		-- do something with summary
		-- baker address available as summary.baker

		local result = {
			-- available in notification message template as <freeSpace>
			freeSpace = tostring(bigint.new(10000)),
			-- available in notification message template as <test>
			test = "testAdditionalData",
		}

		write_response(id, result)
	end,
}

local function listen()
	while true do
		local line = io.read()
		if not line then
			break
		end
		log("request: " .. line)
		local request = hjson.parse(line)
		local error = validate_request(request)
		if error ~= nil then
			write_error(request.id, error)
			return
		end

		local id = request.id
		local method = request.method:sub(#CALL_PREFIX + 1)

		local handler = handlers[method]
		if handler ~= nil then
			handler(request)
		elseif id ~= nil then -- ignores notifications
			write_error(id, METHOD_NOT_FOUND)
		end
	end
end

local ok, error = pcall(listen)
if not ok then
	log("ERROR: " .. tostring(error))
end
