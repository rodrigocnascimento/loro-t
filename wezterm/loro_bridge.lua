local wezterm = require("wezterm")

local M = {}

local function uuid_v4()
  local template = "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx"
  return string.gsub(template, "[xy]", function(c)
    local v = (c == "x") and math.random(0, 0xf) or math.random(8, 0xb)
    return string.format("%x", v)
  end)
end

local function with_backoff(send_fn)
  local backoff_ms = 25
  for attempt = 1, 4 do
    local ok, ack, err = pcall(send_fn)
    if ok and ack and ack.ok then
      return ack, nil
    end

    local reason = (ok and (ack and ack.reason or err)) or tostring(ack)
    if attempt == 4 then
      return nil, reason or "retry_exhausted"
    end

    wezterm.sleep_ms(backoff_ms)
    backoff_ms = math.min(backoff_ms * 2, 150)
  end
end

---Emit event for daemon from WezTerm integration.
---@param send_fn fun(encoded_payload:string):table|nil,string|nil
---@param event table
---@return table|nil,string|nil
function M.emit(send_fn, event)
  event.event_id = event.event_id or uuid_v4()
  local payload = wezterm.json_encode(event)

  return with_backoff(function()
    local ack, err = send_fn(payload)
    if err then
      return nil, err
    end
    if not ack then
      return nil, "missing_ack"
    end
    if ack.ok == false then
      return nil, ack.reason or "daemon_rejected"
    end
    return ack, nil
  end)
end

return M
