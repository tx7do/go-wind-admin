# 认证权鉴中间件

## 介绍

本目录下的中间件用于处理用户认证和权限管理，确保只有经过授权的用户才能访问特定的资源和功能。

## JWT 校验

登录的时候，JWT Token会存一份到Redis，后续每次请求携带Token时，会去Redis校验Token的有效性，防止Token被篡改或伪造。

我们可以使用`WithAccessTokenChecker`和`WithAccessTokenCheckerFunc`来启用对访问令牌的校验。默认这个功能是关闭的。

我们采用的是“白名单”机制来校验JWT Token，虽然这违背了JWT无状态的初衷，但能有效防止Token被篡改或伪造。也可以有效的踢掉某个用户的登录状态。

另外还有一种“黑名单”机制，可以使用`WithAccessTokenBlocker`和`WithAccessTokenBlockerFunc`来启用对访问令牌的阻断。默认这个功能是关闭的。
