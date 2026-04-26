import { useCallback, useEffect, useRef, useState } from 'react';
import { GeocodeSuggestion, suggestAddresses } from '../utils/routeOptimizer';

const DEBOUNCE_MS = 500;

export function useAddressAutocomplete() {
  const [suggestions, setSuggestions] = useState<GeocodeSuggestion[]>([]);
  const [open, setOpen] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancel = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const reset = useCallback(() => {
    cancel();
    setSuggestions([]);
    setOpen(false);
  }, [cancel]);

  const query = useCallback((value: string) => {
    cancel();
    if (!value.trim()) {
      setSuggestions([]);
      setOpen(false);
      return;
    }
    timerRef.current = setTimeout(async () => {
      try {
        const results = await suggestAddresses(value);
        setSuggestions(results);
        setOpen(results.length > 0);
      } catch {
        // тихо игнорируем — автодополнение не критично
      }
    }, DEBOUNCE_MS);
  }, [cancel]);

  useEffect(() => () => cancel(), [cancel]);

  return { suggestions, open, setOpen, query, reset };
}
