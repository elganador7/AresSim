export function formatError(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}

export function reportError(context: string, error: unknown): void {
  console.error(`[${context}] ${formatError(error)}`, error);
}
