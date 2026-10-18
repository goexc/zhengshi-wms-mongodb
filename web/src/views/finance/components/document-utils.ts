export function invoiceTotals(amount: number, rate: number) {
  if (!Number.isFinite(amount) || !Number.isFinite(rate) || amount < 0 || rate < 0) return { tax_amount: 0, total_amount: 0 };
  const cents = BigInt(Math.round(amount * 100));
  const rateHundredths = BigInt(Math.round(rate * 100));
  const taxCents = (cents * rateHundredths + 5000n) / 10000n;
  return { tax_amount: Number(taxCents) / 100, total_amount: Number(cents + taxCents) / 100 };
}

export function operationKey(kind: string): string {
  const random = typeof crypto.randomUUID === 'function' ? crypto.randomUUID() : `${Date.now()}-${crypto.getRandomValues(new Uint32Array(2)).join('-')}`;
  return `${kind}:${random}`;
}

export function dayStart(value: string | number): number {
  if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return Math.floor(Date.parse(`${value}T00:00:00+08:00`) / 1000);
  }
  return Math.floor((Number(value) + 8 * 3600) / 86400) * 86400 - 8 * 3600;
}

export function dayEnd(value: string | number): number {
  return dayStart(value) + 86400 - 1;
}

export function dateInput(value: number): string {
  return new Date((value + 8 * 3600) * 1000).toISOString().slice(0, 10);
}
