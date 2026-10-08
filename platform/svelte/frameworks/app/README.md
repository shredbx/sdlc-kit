# Svelte app framework

`@sbx/app-svelte` provides server-side data primitives for SvelteKit applications. A data module
pairs a runtime schema with a read-only repository, and a per-request registry makes those modules
available to route loaders.

Repositories expose asynchronous `findById` and `findMany` operations. The initial
`createReadOnlyRepository` implementation validates records against the supplied schema whenever
they are read. Applications provide their own schema and repository data source, so checked-in
content and a future API adapter can share a contract without coupling the framework to either.

Use `createDataRegistry` in the application composition root (for example, a SvelteKit server hook)
and resolve registered modules by key in server load functions. Keep data access on the server;
components should receive loaded values as props.
