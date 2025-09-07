## Contributing

1. Fork & branch: `feat/your-feature`
2. Add/update tests (unit + integration) before changing behavior.
3. Run `bun test` – ensure all pass.
4. Keep commits small & focused.
5. Update README if adding env vars or drivers.

### Code Style

- Prefer small, composable DAL methods.
- Use TypeScript strict typing; avoid `any` unless boundary.
- Keep UI components client/server boundaries clear.

### Testing Guidance

- For new DALs: mock external clients; assert query frequency & merging logic.
- Add subscription event tests when emitting new event types.

### PR Checklist

- [ ] Tests added/updated
- [ ] README/CONTRIBUTING updated if needed
- [ ] No unused deps / console noise

Thanks for contributing!
