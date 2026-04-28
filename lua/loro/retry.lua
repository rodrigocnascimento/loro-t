local M = {}

---@class RetryOptions
---@field attempts integer
---@field initial_backoff_ms integer
---@field max_backoff_ms integer
---@field jitter_ms integer

local function sleep_ms(ms)
  vim.wait(ms)
end

---Run a function with short exponential backoff.
---@generic T
---@param fn fun():T|nil, string|nil
---@param opts RetryOptions|nil
---@return T|nil, string|nil
function M.with_backoff(fn, opts)
  opts = opts or {}
  local attempts = opts.attempts or 3
  local backoff = opts.initial_backoff_ms or 25
  local max_backoff = opts.max_backoff_ms or 200
  local jitter_ms = opts.jitter_ms or 10

  local last_err
  for i = 1, attempts do
    local ok, result, err = pcall(fn)
    if ok and err == nil then
      return result, nil
    end

    last_err = ok and err or result
    if i < attempts then
      local jitter = math.random(0, jitter_ms)
      sleep_ms(backoff + jitter)
      backoff = math.min(backoff * 2, max_backoff)
    end
  end

  return nil, last_err or "retry_exhausted"
end

return M
