local retry = require("loro.retry")

local M = {}

---@class TransportConfig
---@field socket_path string
---@field max_buffer_size integer
---@field enable_offline_buffer boolean
---@field logger fun(level:string, message:string, ctx:table|nil)

---@class Ack
---@field ok boolean
---@field reason string|nil

local default_config = {
  socket_path = "/tmp/loro.sock",
  max_buffer_size = 100,
  enable_offline_buffer = true,
}

local function default_logger(level, message, ctx)
  vim.notify(string.format("[loro:%s] %s", level, message), vim.log.levels.INFO)
  if ctx then
    vim.notify(vim.inspect(ctx), vim.log.levels.DEBUG)
  end
end

local function uuid_v4()
  local template = "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
  return string.gsub(template, "[xy]", function(c)
    local v = (c == "x") and math.random(0, 0xf) or math.random(8, 0xb)
    return string.format("%x", v)
  end)
end

local Transport = {}
Transport.__index = Transport

function Transport.new(config)
  local merged = vim.tbl_extend("force", default_config, config or {})
  merged.logger = merged.logger or default_logger

  return setmetatable({
    config = merged,
    offline_buffer = {},
  }, Transport)
end

function Transport:next_event_id()
  return uuid_v4()
end

function Transport:push_offline(event)
  if not self.config.enable_offline_buffer then
    self.config.logger("warn", "event_discarded_offline_disabled", { event_id = event.event_id })
    return
  end

  table.insert(self.offline_buffer, event)
  if #self.offline_buffer > self.config.max_buffer_size then
    local removed = table.remove(self.offline_buffer, 1)
    self.config.logger("warn", "event_discarded_buffer_limit", {
      dropped_event_id = removed.event_id,
      max_buffer_size = self.config.max_buffer_size,
    })
  end
end

function Transport:flush_buffer()
  if #self.offline_buffer == 0 then
    return
  end

  local pending = self.offline_buffer
  self.offline_buffer = {}
  for _, event in ipairs(pending) do
    local _, err = self:send_event(event)
    if err then
      self:push_offline(event)
      break
    end
  end
end

function Transport:_send_raw(payload)
  local sock = vim.uv.new_pipe(false)
  local connected, connect_err = pcall(sock.connect, sock, self.config.socket_path)
  if not connected then
    sock:close()
    return nil, connect_err
  end

  sock:write(payload .. "\n")
  local response = ""
  sock:read_start(function(err, chunk)
    if err then
      response = vim.json.encode({ ok = false, reason = err })
      return
    end
    if chunk then
      response = response .. chunk
      return
    end
    sock:read_stop()
    sock:close()
  end)

  vim.wait(250, function()
    return response ~= ""
  end)

  if response == "" then
    return nil, "ack_timeout"
  end

  local ack = vim.json.decode(response)
  if type(ack) ~= "table" then
    return nil, "invalid_ack"
  end

  if not ack.ok then
    return nil, ack.reason or "daemon_rejected"
  end

  return ack, nil
end

function Transport:send_event(event)
  event.event_id = event.event_id or self:next_event_id()
  local payload = vim.json.encode(event)

  local ack, err = retry.with_backoff(function()
    return self:_send_raw(payload)
  end, {
    attempts = 4,
    initial_backoff_ms = 25,
    max_backoff_ms = 150,
  })

  if err then
    self.config.logger("warn", "send_failed_buffering", { reason = err, event_id = event.event_id })
    self:push_offline(event)
    return nil, err
  end

  return ack, nil
end

M.Transport = Transport

return M
