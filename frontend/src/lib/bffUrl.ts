const base = import.meta.env.VITE_BFF_URL ?? ''

export const bffUrl = (path: string) => `${base}${path}`
