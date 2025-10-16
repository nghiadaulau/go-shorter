# UI (Next.js + Tailwind) Scaffold

- Create app with create-next-app in this folder:

```bash
npm create next-app@latest . -- --ts --eslint
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

- Add `.env.example` with:

```
NEXT_PUBLIC_API_BASE=http://localhost:8080
```

- Pages to implement: Login, Dashboard, Create Link, Link List, Link Detail, Stats.
