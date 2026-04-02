// node pkce.js
const crypto = require('crypto')
const verifier = crypto.randomBytes(32).toString('base64url')
const challenge = crypto.createHash('sha256').update(verifier).digest('base64url')
console.log('code_verifier:', verifier)
console.log('code_challenge:', challenge)
