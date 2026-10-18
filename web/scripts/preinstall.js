if (!/^pnpm\//.test(process.env.npm_config_user_agent || '')) {
    console.error('Please use pnpm install --frozen-lockfile in the web directory.')
    process.exit(1)
}
