// with lazy vim
return {
	{
		"neovim/nvim-lspconfig",
		opts = function()
			vim.api.nvim_create_autocmd("FileType", {
				pattern = "graphql",
				callback = function()
					local client_id = vim.lsp.start({
						name = "houdini_lsp",
						cmd = { "/home/d2du/code/oss/houdini/houdini-lib/packages/lsp/main" },
					})

					if not client_id then
						vim.notify("houdini_lsp failed to start", vim.log.levels.ERROR)
					end
				end,
			})
		end,
	},
}

