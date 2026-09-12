const cache = new Map<string, HTMLImageElement>()
const inflight = new Map<string, Promise<HTMLImageElement>>()

const ASSET_PROXY_HOSTS = [
  "github.com",
  "user-images.githubusercontent.com",
  "private-user-images.githubusercontent.com",
]

export function assetSrc(src: string): string {
  try {
    if (!ASSET_PROXY_HOSTS.includes(new URL(src).host)) return src
  } catch {
    return src
  }
  return `/api/asset?url=${encodeURIComponent(src)}`
}

export function preloadImage(url: string): Promise<HTMLImageElement> {
  const done = cache.get(url)
  if (done) return Promise.resolve(done)

  const pending = inflight.get(url)
  if (pending) return pending

  const img = new Image()
  const promise = new Promise<HTMLImageElement>((resolve, reject) => {
    img.onload = () => resolve(img)
    img.onerror = reject
    img.src = url
  })
    .then(async () => {
      await img.decode().catch(() => {})
      cache.set(url, img)
      return img
    })
    .finally(() => {
      inflight.delete(url)
    })

  inflight.set(url, promise)
  return promise
}

export function preloadImages(urls: string[]) {
  return Promise.allSettled(urls.map(preloadImage))
}

export function readyImage(url: string): HTMLImageElement | undefined {
  const img = cache.get(url)
  if (!img) return undefined
  return img.isConnected ? (img.cloneNode(true) as HTMLImageElement) : img
}

const IMAGE_PATTERN = /!\[[^\]]*\]\(\s*<?([^)\s>]+)>?(?:\s+["'][^"']*["'])?\s*\)|<img[^>]+src=["']([^"']+)["']/gi

export function markdownImageUrls(content: string): string[] {
  const urls = new Set<string>()
  for (const match of content.matchAll(IMAGE_PATTERN)) {
    const src = match[1] ?? match[2]
    if (src) urls.add(assetSrc(src))
  }
  return [...urls]
}

export function preloadMarkdownImages(content: string | undefined) {
  if (!content) return
  void preloadImages(markdownImageUrls(content))
}
