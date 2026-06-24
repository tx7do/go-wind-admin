-- Example: Self-registering script
-- This script registers itself to a hook when loaded

-- Configuration
local HOOK_NAME = "user.created"
local SCRIPT_NAME = "welcome_email_sender"

-- 1. Register the hook (if it doesn't exist, this is safe to call multiple times)
hook.register(HOOK_NAME, "Triggered when a new user is created")

-- 2. Register this script to the hook
hook.add_script(HOOK_NAME, {
    name = SCRIPT_NAME,
    source = [[
        function execute(ctx)
            -- Get user data from context
            local user_id = ctx.get("user_id")
            local email = ctx.get("email")
            local username = ctx.get("username")

            log.info("Processing new user: " .. username)

            -- Send welcome email via eventbus
            eventbus.publish("email.send", {
                to = email,
                subject = "Welcome to our platform!",
                template = "welcome",
                data = {
                    username = username,
                    user_id = user_id
                }
            }, "email")

            log.info("Welcome email queued for: " .. email)

            -- Store in cache for tracking
            cache.set("welcome_sent:" .. user_id, "true", 86400)

            return true
        end
    ]],
    enabled = true,
    priority = 10,
    description = "Sends welcome email to newly registered users"
})

log.info("Script registered successfully: " .. SCRIPT_NAME .. " -> " .. HOOK_NAME)

-- 3. List all hooks to verify
local all_hooks = hook.list()
log.info("Total hooks registered: " .. #all_hooks)
