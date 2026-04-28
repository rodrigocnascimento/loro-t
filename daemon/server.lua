local dedup = require("daemon.dedup")

local M = {}

local Server = {}
Server.__index = Server

function Server.new(opts)
  opts = opts or {}
  local cache = dedup.DedupCache.new(opts.dedup_cache_size or 500)
  return setmetatable({
    cache = cache,
    handle_event = opts.handle_event,
  }, Server)
end

function Server:process_line(raw_line)
  local ok, event = pcall(vim.json.decode, raw_line)
  if not ok then
    return vim.json.encode({ ok = false, reason = "invalid_json" })
  end

  if type(event) ~= "table" then
    return vim.json.encode({ ok = false, reason = "invalid_event" })
  end

  if type(event.event_id) ~= "string" or event.event_id == "" then
    return vim.json.encode({ ok = false, reason = "missing_event_id" })
  end

  if self.cache:has(event.event_id) then
    return vim.json.encode({ ok = true, reason = "duplicate_ignored", event_id = event.event_id })
  end

  local accepted, reason = self.handle_event(event)
  if not accepted then
    return vim.json.encode({ ok = false, reason = reason or "handler_rejected", event_id = event.event_id })
  end

  self.cache:add(event.event_id)
  return vim.json.encode({ ok = true, reason = "accepted", event_id = event.event_id })
end

M.Server = Server
return M
