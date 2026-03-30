export interface ModelAssetSpec {
  uri: string;
  scale: number;
  minimumPixelSize: number;
  maximumScale?: number;
  closeDisplayM?: number;
}

const MODEL_ASSETS: Record<string, ModelAssetSpec> = {};

export function modelAssetFor(visualModelId?: string): ModelAssetSpec | undefined {
  if (!visualModelId) {
    return undefined;
  }
  return MODEL_ASSETS[visualModelId.trim().toLowerCase()];
}
