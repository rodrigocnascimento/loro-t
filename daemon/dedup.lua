local M = {}

local DedupCache = {}
DedupCache.__index = DedupCache

function DedupCache.new(max_items)
  return setmetatable({
    max_items = max_items or 500,
    queue = {},
    seen = {},
  }, DedupCache)
end

function DedupCache:has(event_id)
  return self.seen[event_id] == true
end

function DedupCache:add(event_id)
  if self.seen[event_id] then
    return
  end

  table.insert(self.queue, event_id)
  self.seen[event_id] = true

  if #self.queue > self.max_items then
    local removed = table.remove(self.queue, 1)
    self.seen[removed] = nil
  end
end

M.DedupCache = DedupCache
return M
