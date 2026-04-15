<!--
Copyright (c) 2026, SoftlaneIT (https://softlaneit.com/) All Rights Reserved.

SoftlaneIT licenses this file to you under the Apache License,
Version 2.0 (the "LICENSE"); you may not use this file except
in compliance with the LICENSE.
You may obtain a copy of the LICENSE at

https://softlaneit.com/LICENSE.txt

Unless required by applicable law or agreed to in writing,
software distributed under the LICENSE is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the LICENSE for the
specific language governing permissions and limitations
under the LICENSE.
-->

# Contributing to ServiceForge

Thank you for your interest in contributing.

## Development Workflow

1. Fork and clone the repository.
2. Create a feature branch from main.
3. Keep changes focused and small.
4. Add or update tests where applicable.
5. Submit a pull request with clear context and impact.

## Pull Request Requirements

- Explain problem, solution, and risk.
- Reference related issue numbers.
- Include test evidence (commands and output summary).
- Keep commit history readable.

## Code Standards

- Go code should compile across all service modules.
- Public APIs should be versioned and documented.
- New modules must preserve tenant context propagation.
- Use structured logging and avoid sensitive data in logs.

## Local Validation

Run these before opening a PR:

```bash
make up
make auth
make tenant
make config
make booking
make gateway
```

For UI:

```bash
cd apps/management-ui
npm install
npm run build
```

## Licensing

By contributing to this repository, you agree that your contributions are licensed under Apache License 2.0.
