/**
 * Model types a global model can have. `storedKey` is the canonical name the
 * server returns; `requestKey` is what the page sends for a default, because
 * the admin request client snake-cases top-level keys and "speech2text" would
 * become "speech_2_text". The server accepts both names.
 */
export const GlobalModelTypes = [
  { storedKey: 'chat', requestKey: 'chat' },
  { storedKey: 'embedding', requestKey: 'embedding' },
  { storedKey: 'rerank', requestKey: 'rerank' },
  { storedKey: 'speech2text', requestKey: 'asr' },
  { storedKey: 'image2text', requestKey: 'vision' },
  { storedKey: 'tts', requestKey: 'tts' },
  { storedKey: 'ocr', requestKey: 'ocr' },
  { storedKey: 'decision', requestKey: 'decision' },
] as const;

export type GlobalModelTypeKey = (typeof GlobalModelTypes)[number]['storedKey'];

export const DefaultMaxTokens = 8192;
