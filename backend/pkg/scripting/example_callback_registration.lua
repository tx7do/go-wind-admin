-- Example: Callback-based hook registration
-- This demonstrates the simplified callback API

-- Simple callback registration
-- Syntax: hook.register(hook_name, description, callback_function)
hook.register("user.login", "Triggered when a user logs in", function(ctx)
    local user_id = ctx.get("user_id")
    local ip = ctx.get("ip_address")
    local username = ctx.get("username")

    log.info("User logged in: " .. username .. " from " .. ip)

    -- Log to cache for rate limiting
    local login_key = "login_count:" .. user_id
    cache.incr(login_key, 1)
    cache.expire(login_key, 3600)

    return true
end)

-- Minimal registration (without description)
hook.register("data.validated", nil, function(ctx)
    local data = ctx.get("data")
    log.info("Data validated: " .. data)
    ctx.set("validation_timestamp", os.time())
    return true
end)

-- Callback with early abort
hook.register("payment.process", "Process payment", function(ctx)
    local amount = ctx.get("amount")
    local user_id = ctx.get("user_id")

    -- Check if amount is valid
    if amount <= 0 then
        log.error("Invalid payment amount: " .. amount)
        ctx.stop("Invalid amount")
        return false  -- Abort processing
    end

    -- Check user balance
    local balance = cache.get("balance:" .. user_id)
    if not balance or balance < amount then
        log.error("Insufficient balance for user: " .. user_id)
        ctx.stop("Insufficient balance")
        return false  -- Abort processing
    end

    -- Process payment
    cache.decr("balance:" .. user_id, amount)
    log.info("Payment processed: $" .. amount .. " for user " .. user_id)

    return true
end)

-- Callback with eventbus integration
hook.register("order.created", "New order created", function(ctx)
    local order_id = ctx.get("order_id")
    local customer_email = ctx.get("customer_email")
    local total = ctx.get("total")

    -- Publish to eventbus for async processing
    eventbus.publish("email.send", {
        to = customer_email,
        subject = "Order Confirmation #" .. order_id,
        template = "order_confirmation",
        data = {
            order_id = order_id,
            total = total
        }
    }, "email")

    -- Update inventory
    eventbus.publish("inventory.update", {
        order_id = order_id,
        action = "reserve"
    }, "warehouse")

    log.info("Order " .. order_id .. " processed, notifications sent")

    return true
end)

-- Multiple callbacks and scripts can coexist
-- First, register a callback
hook.register("user.updated", "User profile updated", function(ctx)
    local user_id = ctx.get("user_id")
    log.info("Callback: User profile updated for user " .. user_id)
    ctx.set("callback_executed", true)
    return true
end)

-- Then add additional scripts to the same hook
hook.add_script("user.updated", {
    name = "audit_log",
    source = [[
        function execute(ctx)
            local user_id = ctx.get("user_id")
            local changes = ctx.get("changes")
            log.info("Audit: Recording changes for user " .. user_id)
            -- Both callback and this script will execute
            return true
        end
    ]],
    priority = 5,
    enabled = true,
    description = "Audit trail for user updates"
})

log.info("Callback-based hooks registered successfully")
