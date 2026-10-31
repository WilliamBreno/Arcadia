import type { ReactNode } from 'react'

// Item 11.7: ícones das coleções e temas. Desenho próprio da Evve em linha
// (traço na cor do texto) com um detalhe no verde da marca. As chaves
// precisam bater com domain.IconesColecao no backend.

const A = 'var(--marca)' // detalhe em verde

const desenhos: Record<string, ReactNode> = {
  estrela: (
    <>
      <path d="M24 8l4.7 9.6 10.6 1.5-7.7 7.5 1.8 10.5L24 32.1l-9.4 5 1.8-10.5-7.7-7.5 10.6-1.5z" />
      <path d="M38 6v6M35 9h6" stroke={A} />
    </>
  ),
  abobora: (
    <>
      <path d="M24 14c-10 0-16 6-16 13.5S14 40 24 40s16-5 16-12.5S34 14 24 14z" />
      <path d="M15 18.5c-2.5 3-3 15 0 18.5M33 18.5c2.5 3 3 15 0 18.5" />
      <path d="M24 14c0-3 1.5-5 4.5-6.5" stroke={A} />
      <path d="M17.5 26l2.5-3.5 2.5 3.5zM25.5 26l2.5-3.5 2.5 3.5z" />
      <path d="M17.5 31.5l2 1.5 2-1.5 2.5 1.5 2.5-1.5 2 1.5 2-1.5" />
    </>
  ),
  arvore: (
    <>
      <path d="M24 9l-8 11h4l-7 9h5l-7 9h26l-7-9h5l-7-9h4z" />
      <path d="M24 38v4" />
      <path d="M24 4.5l1.2 2.5 2.8.4-2 2 .5 2.8-2.5-1.3-2.5 1.3.5-2.8-2-2 2.8-.4z" stroke={A} />
    </>
  ),
  tacas: (
    <>
      <path d="M13 12l8 2-2.5 9.5a4 4 0 01-7.7-2zM18.5 24l-2.5 10M12 33l8 2" />
      <path d="M35 12l-8 2 2.5 9.5a4 4 0 007.7-2zM29.5 24l2.5 10M36 33l-8 2" />
      <path d="M24 5v4M20.5 7.5l1.5 1.5M27.5 7.5L26 9" stroke={A} />
    </>
  ),
  sol: (
    <>
      <circle cx="24" cy="24" r="8" />
      <path d="M24 6v5M24 37v5M6 24h5M37 24h5M11.3 11.3l3.5 3.5M33.2 33.2l3.5 3.5M11.3 36.7l3.5-3.5M33.2 14.8l3.5-3.5" />
      <path d="M28.5 20a6 6 0 012 4.5" stroke={A} />
    </>
  ),
  mascara: (
    <>
      <path d="M7 19c4-2 10-1.5 13 1.5 2.5 2.5 5.5 2.5 8 0 3-3 9-3.5 13-1.5-.5 7-4 12-9.5 12-3.5 0-5-2.5-7.5-2.5S20 32 16.5 32C11 32 7.5 26 7 19z" />
      <path d="M13 23.5c1.5-1 3.5-1 5 0M30 23.5c1.5-1 3.5-1 5 0" />
      <path d="M36 17c1-4 3.5-7 7-8M33 16c0-3.5 1.5-6.5 4.5-8.5" stroke={A} />
    </>
  ),
  fogueira: (
    <>
      <path d="M10 40l28-7M38 40l-28-7" />
      <path d="M24 31c-5 0-7.5-3.5-7.5-7 0-5 5-7 5.5-13 3.5 2.5 9.5 7 9.5 13 0 3.5-2.5 7-7.5 7z" stroke={A} />
      <path d="M24 31c-2 0-3-1.5-3-3 0-2 2-3 2.5-5 1.5 1 3.5 3 3.5 5s-1 3-3 3z" />
    </>
  ),
  pipa: (
    <>
      <path d="M26 5l12 13-12 17-12-17z" />
      <path d="M26 5v30M14 18h24" />
      <path d="M26 35c-2 3-1 5 1 6.5s2 3.5 0 5" stroke={A} />
      <path d="M24.5 39.5l3 -1M25 44l3-1" stroke={A} />
    </>
  ),
  controle: (
    <>
      <path d="M15 15h18c5 0 8 4 9 10l1 6c.5 3.5-3 5.5-5.5 3L33 29H15l-4.5 5c-2.5 2.5-6 .5-5.5-3l1-6c1-6 4-10 9-10z" />
      <path d="M15 19.5v6M12 22.5h6" />
      <circle cx="31" cy="20.5" r="1.5" stroke={A} />
      <circle cx="35" cy="24.5" r="1.5" stroke={A} />
    </>
  ),
  ingresso: (
    <>
      <path d="M8 15h32v6a3 3 0 000 6v6H8v-6a3 3 0 000-6z" />
      <path d="M17 15v18" strokeDasharray="2 3" stroke={A} />
      <path d="M24 28l9-8" />
      <circle cx="25" cy="20.5" r="1.5" />
      <circle cx="32" cy="27.5" r="1.5" />
    </>
  ),
  calendario: (
    <>
      <rect x="8" y="11" width="32" height="29" rx="4" />
      <path d="M8 19h32M16 7v7M32 7v7" />
      <rect x="25" y="26" width="8" height="7" rx="1.5" stroke={A} />
      <path d="M15 26h4M15 32h4" />
    </>
  ),
  tela: (
    <>
      <rect x="9" y="11" width="30" height="20" rx="2.5" />
      <path d="M5 36h38l-2 3H7z" />
      <path d="M21 16.5v9l7.5-4.5z" stroke={A} />
    </>
  ),
  microfone: (
    <>
      <rect x="13" y="6" width="12" height="18" rx="6" />
      <path d="M13 13h4M13 17h4" />
      <path d="M9 20a10 10 0 0020 0M19 30v7M13 41h12" />
      <path d="M32 12a6 6 0 010 8M36 9a11 11 0 010 14" stroke={A} />
    </>
  ),
  nota: (
    <>
      <path d="M18 34V12l18-4v22" />
      <circle cx="14" cy="34" r="4" />
      <circle cx="32" cy="30" r="4" />
      <path d="M18 18l18-4" stroke={A} />
    </>
  ),
  teatro: (
    <>
      <path d="M8 10h16v11c0 6-3.5 10-8 10s-8-4-8-10z" />
      <path d="M12 17h2M18 17h2M12.5 24c2 2 5 2 7 0" />
      <path d="M25 18h15v10c0 6-3.5 10-7.5 10S25 34 25 28z" stroke={A} />
      <path d="M29 24h2M34 24h2M29.5 33c2-2 4.5-2 6.5 0" stroke={A} />
    </>
  ),
  danca: (
    <>
      <circle cx="25" cy="9" r="3.5" />
      <path d="M25 13l-2 11 6 6-2 11M23 24l-7 4-3 9M24 16l-9-5M25 16l8-6" />
      <path d="M36 22c2 1.5 3 3.5 3 6M10 18c-2 1.5-3 3.5-3 6" stroke={A} />
    </>
  ),
  cosplay: (
    <>
      <path d="M24 6c-3 6-7 18-10 26h20C31 24 27 12 24 6z" />
      <path d="M8 34c5 3 27 3 32 0" />
      <path d="M14 32c-4 1-6 2-6 2M34 32c4 1 6 2 6 2" />
      <path d="M24 20l1 2.2 2.4.3-1.8 1.6.5 2.4-2.1-1.2-2.1 1.2.5-2.4-1.8-1.6 2.4-.3z" stroke={A} />
    </>
  ),
  comida: (
    <>
      <path d="M9 22c0-7 7-11 15-11s15 4 15 11z" />
      <path d="M8 27h32M10 32h28a3 3 0 01-3 4H13a3 3 0 01-3-4z" />
      <path d="M8 27c3-2 5 2 8 0s5 2 8 0 5 2 8 0 5 2 8 0" stroke={A} />
      <path d="M18 16.5h.01M24 15h.01M30 16.5h.01" />
    </>
  ),
  livro: (
    <>
      <path d="M24 13c-4-3-10-4-16-3v26c6-1 12 0 16 3 4-3 10-4 16-3V10c-6-1-12 0-16 3zM24 13v26" />
      <path d="M31 11v10l3-2 3 2V10.5" stroke={A} />
    </>
  ),
  coracao: (
    <>
      <path d="M24 38S10 30 10 20a7 7 0 0114-3 7 7 0 0114 3c0 10-14 18-14 18z" />
      <path d="M24 6v3M38 9l-2 2M10 9l2 2M43 22h-3M5 22h3" stroke={A} />
    </>
  ),
}

export const iconesColecao = Object.keys(desenhos)

export function IconeColecao({ nome, className = 'size-12' }: { nome: string; className?: string }) {
  return (
    <svg
      viewBox="0 0 48 48"
      fill="none"
      stroke="currentColor"
      strokeWidth={2}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      {desenhos[nome] ?? desenhos.estrela}
    </svg>
  )
}
