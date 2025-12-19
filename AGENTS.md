# Agent Instructions

## Git Branch Naming

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#specification) types for branch names and commit messages.

### Format

**Basic:** `<type>/<description>`  
**With scope:** `<type>(<scope>)/<description>`  
**Breaking change:** `<type>!: <description>`

### Rules

- **kebab-case** descriptions (50 chars max)
- Present tense ("add", not "added")
- Be concise but clear

### Types

- `feat`: New feature
- `fix`: Bug fix  
- `docs`: Documentation
- `style`: Formatting only
- `refactor`: Code improvement
- `perf`: Performance boost
- `test`: Tests
- `build`: Build system
- `ci`: CI configuration
- `chore`: Other changes
- `revert`: Previous commit

### Scopes (Optional)

**Common scopes:**
- `auth` - Authentication
- `ui` - User interface
- `api` - API endpoints
- `config` - Configuration
- `deps` - Dependencies
- `test` - Testing

### Examples

**Basic:**
- `feat/user-login`
- `fix/memory-leak`
- `docs/update-readme`

**Scoped:**
- `feat(auth)/oauth-support`
- `fix(ui)/mobile-layout`
- `docs(api)/endpoints`

**Breaking:**
- `feat!: remove-deprecated-api`

### Workflow

1. Create branch per logical work unit
2. Make focused commits
3. Follow commit conventions
4. Push and PR
5. Delete after merge

### Mistakes to Avoid

❌ `feat/UserLogin` (caps)  
❌ `fix/memory_leak` (underscores)  
❌ `feature/login` (wrong type)  
❌ `feat/very-long-description-exceeding-limit`  
❌ `fix/` (no description)  

✅ `feat/user-login`  
✅ `fix/memory-leak`  
✅ `feat(auth)/oauth`
