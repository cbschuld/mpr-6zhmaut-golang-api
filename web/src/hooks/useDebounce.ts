import { useRef, useCallback } from "react";

export function useDebounce<T extends (...args: never[]) => void>(
  fn: T,
  delayMs: number
): T {
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  return useCallback(
    ((...args: Parameters<T>) => {
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => fn(...args), delayMs);
    }) as T,
    [fn, delayMs]
  );
}
