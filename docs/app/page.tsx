import Link from 'next/link';

export default function HomePage() {
  return (
    <main className="flex flex-1 flex-col">
      {/* Hero Section */}
      <section className="relative overflow-hidden border-b border-fd-border bg-fd-background py-20 md:py-32">
        <div className="absolute inset-0 bg-gradient-to-br from-fd-primary/5 via-transparent to-fd-primary/5" />
        <div className="container relative mx-auto px-4 text-center">
          <div className="mx-auto max-w-3xl">
            <h1 className="mb-6 text-4xl font-bold tracking-tight text-fd-foreground md:text-6xl">
              Lux SDK
            </h1>
            <p className="mb-8 text-lg text-fd-muted-foreground md:text-xl">
              The official TypeScript/JavaScript SDK for building on Lux Network.
              Create wallets, sign transactions, interact with smart contracts, and
              build cross-chain applications with ease.
            </p>
            <div className="flex flex-col items-center justify-center gap-4 sm:flex-row">
              <Link
                href="/docs"
                className="inline-flex h-11 items-center justify-center rounded-lg bg-fd-primary px-8 text-sm font-medium text-fd-primary-foreground transition-colors hover:bg-fd-primary/90"
              >
                Get Started
              </Link>
              <Link
                href="https://github.com/luxfi/sdk"
                className="inline-flex h-11 items-center justify-center rounded-lg border border-fd-border bg-fd-card px-8 text-sm font-medium text-fd-foreground transition-colors hover:bg-fd-accent"
              >
                View on GitHub
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Quick Install Section */}
      <section className="border-b border-fd-border bg-fd-card py-12">
        <div className="container mx-auto px-4">
          <div className="mx-auto max-w-2xl text-center">
            <h2 className="mb-4 text-sm font-semibold uppercase tracking-wider text-fd-muted-foreground">
              Quick Install
            </h2>
            <div className="group relative overflow-hidden rounded-lg border border-fd-border bg-fd-background p-4">
              <code className="font-mono text-sm text-fd-foreground md:text-base">
                npm install @luxfi/sdk
              </code>
              <button
                className="absolute right-3 top-1/2 -translate-y-1/2 rounded-md p-2 text-fd-muted-foreground opacity-0 transition-opacity hover:bg-fd-accent hover:text-fd-foreground group-hover:opacity-100"
                aria-label="Copy to clipboard"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="16"
                  height="16"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
                  <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Features Grid */}
      <section className="bg-fd-background py-16 md:py-24">
        <div className="container mx-auto px-4">
          <div className="mb-12 text-center">
            <h2 className="mb-4 text-3xl font-bold tracking-tight text-fd-foreground">
              Everything you need to build on Lux
            </h2>
            <p className="mx-auto max-w-2xl text-fd-muted-foreground">
              A comprehensive toolkit for interacting with all Lux chains including
              P-Chain, X-Chain, C-Chain, and custom L1 networks.
            </p>
          </div>
          <div className="mx-auto grid max-w-5xl gap-6 md:grid-cols-2 lg:grid-cols-3">
            {/* Transactions */}
            <div className="group rounded-xl border border-fd-border bg-fd-card p-6 transition-colors hover:border-fd-primary/50">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-fd-primary/10 text-fd-primary">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M12 2v20M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
                </svg>
              </div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground">
                Transactions
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                Build, sign, and broadcast transactions across P-Chain, X-Chain,
                and C-Chain with type-safe builders.
              </p>
            </div>

            {/* Wallets */}
            <div className="group rounded-xl border border-fd-border bg-fd-card p-6 transition-colors hover:border-fd-primary/50">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-fd-primary/10 text-fd-primary">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M19 7V4a1 1 0 0 0-1-1H5a2 2 0 0 0 0 4h15a1 1 0 0 1 1 1v4h-3a2 2 0 0 0 0 4h3a1 1 0 0 0 1-1v-2a1 1 0 0 0-1-1" />
                  <path d="M3 5v14a2 2 0 0 0 2 2h15a1 1 0 0 0 1-1v-4" />
                </svg>
              </div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground">
                Wallets
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                HD wallet generation with BIP-39/44 support. Manage keys securely
                with mnemonic phrases and derivation paths.
              </p>
            </div>

            {/* Networks */}
            <div className="group rounded-xl border border-fd-border bg-fd-card p-6 transition-colors hover:border-fd-primary/50">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-fd-primary/10 text-fd-primary">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20" />
                  <path d="M2 12h20" />
                </svg>
              </div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground">
                Networks
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                Connect to Mainnet, Testnet, or custom networks. Full support for
                L1 networks and cross-chain operations via Warp.
              </p>
            </div>

            {/* Contracts */}
            <div className="group rounded-xl border border-fd-border bg-fd-card p-6 transition-colors hover:border-fd-primary/50">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-fd-primary/10 text-fd-primary">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
                  <polyline points="14 2 14 8 20 8" />
                  <path d="m10 13-2 2 2 2" />
                  <path d="m14 17 2-2-2-2" />
                </svg>
              </div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground">
                Contracts
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                Interact with smart contracts on C-Chain and EVM L1 chains. Deploy,
                call, and encode/decode contract data.
              </p>
            </div>

            {/* Signing */}
            <div className="group rounded-xl border border-fd-border bg-fd-card p-6 transition-colors hover:border-fd-primary/50">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-fd-primary/10 text-fd-primary">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
                  <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                </svg>
              </div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground">
                Signing
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                Sign messages and transactions with secp256k1. Support for EIP-712
                typed data and personal signatures.
              </p>
            </div>

            {/* Utilities */}
            <div className="group rounded-xl border border-fd-border bg-fd-card p-6 transition-colors hover:border-fd-primary/50">
              <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-fd-primary/10 text-fd-primary">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                >
                  <path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" />
                </svg>
              </div>
              <h3 className="mb-2 text-lg font-semibold text-fd-foreground">
                Utilities
              </h3>
              <p className="text-sm text-fd-muted-foreground">
                Address conversion, bech32 encoding, unit formatting, and more.
                Everything you need for Lux development.
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="border-t border-fd-border bg-fd-card py-16">
        <div className="container mx-auto px-4 text-center">
          <h2 className="mb-4 text-2xl font-bold text-fd-foreground">
            Ready to build?
          </h2>
          <p className="mb-8 text-fd-muted-foreground">
            Check out our guides and API reference to get started.
          </p>
          <div className="flex flex-col items-center justify-center gap-4 sm:flex-row">
            <Link
              href="/docs/getting-started"
              className="inline-flex h-11 items-center justify-center rounded-lg bg-fd-primary px-8 text-sm font-medium text-fd-primary-foreground transition-colors hover:bg-fd-primary/90"
            >
              Read the Docs
            </Link>
            <Link
              href="/docs/api"
              className="inline-flex h-11 items-center justify-center rounded-lg border border-fd-border bg-fd-background px-8 text-sm font-medium text-fd-foreground transition-colors hover:bg-fd-accent"
            >
              API Reference
            </Link>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-fd-border bg-fd-background py-12">
        <div className="container mx-auto px-4">
          <div className="flex flex-col items-center justify-between gap-6 md:flex-row">
            <div className="flex items-center gap-2">
              <span className="text-lg font-semibold text-fd-foreground">Lux SDK</span>
              <span className="text-sm text-fd-muted-foreground">by Lux Partners</span>
            </div>
            <div className="flex items-center gap-6">
              <Link
                href="https://github.com/luxfi/sdk"
                className="text-sm text-fd-muted-foreground transition-colors hover:text-fd-foreground"
              >
                GitHub
              </Link>
              <Link
                href="https://lux.network"
                className="text-sm text-fd-muted-foreground transition-colors hover:text-fd-foreground"
              >
                Lux Network
              </Link>
              <Link
                href="https://discord.gg/luxdefi"
                className="text-sm text-fd-muted-foreground transition-colors hover:text-fd-foreground"
              >
                Discord
              </Link>
              <Link
                href="https://twitter.com/luxdefi"
                className="text-sm text-fd-muted-foreground transition-colors hover:text-fd-foreground"
              >
                Twitter
              </Link>
            </div>
          </div>
          <div className="mt-8 text-center text-sm text-fd-muted-foreground">
            Released under the MIT License. Copyright 2025 Lux Partners.
          </div>
        </div>
      </footer>
    </main>
  );
}
