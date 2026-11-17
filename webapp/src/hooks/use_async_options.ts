import {useCallback, useEffect, useMemo, useRef, useState} from 'react';

import type {ComboboxOption} from '@mattermost/compass-ui/components/combobox';

const DEBOUNCE_MS = 150;

type Params<T> = {

    // Runs the search for a term and resolves with the matching items.
    search: (term: string) => Promise<T[]>;
    getValue: (item: T) => string;
    getLabel: (item: T) => string;

    // Items that are already selected, so their chips keep a label.
    selected: T[];
    errorMessage: string;
};

// Backs a Combobox whose options come from a server search. It debounces the search, drops stale
// responses and remembers every item it has loaded, so a selection can be mapped back to its item.
export default function useAsyncOptions<T>({search, getValue, getLabel, selected, errorMessage}: Params<T>) {
    const [loaded, setLoaded] = useState<T[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');
    const itemsById = useRef(new Map<string, T>());
    const requestId = useRef(0);
    const timer = useRef<ReturnType<typeof setTimeout>>();

    useEffect(() => {
        return () => clearTimeout(timer.current);
    }, []);

    const remember = useCallback((items: T[]) => {
        items.forEach((item) => itemsById.current.set(getValue(item), item));
    }, [getValue]);

    remember(selected);

    const onInputChange = useCallback((term: string) => {
        clearTimeout(timer.current);
        const id = ++requestId.current;
        setLoading(true);

        timer.current = setTimeout(async () => {
            try {
                const items = await search(term);
                if (id !== requestId.current) {
                    return;
                }
                remember(items);
                setLoaded(items);
                setError('');
            } catch {
                if (id !== requestId.current) {
                    return;
                }
                setLoaded([]);
                setError(errorMessage);
            } finally {
                if (id === requestId.current) {
                    setLoading(false);
                }
            }
        }, DEBOUNCE_MS);
    }, [search, remember, errorMessage]);

    const selectedValues = useMemo(() => selected.map(getValue), [selected, getValue]);

    const options = useMemo<ComboboxOption[]>(() => {
        return loaded.
            filter((item) => !selectedValues.includes(getValue(item))).
            map((item) => ({value: getValue(item), label: getLabel(item)}));
    }, [loaded, selectedValues, getValue, getLabel]);

    const selectedOptions = useMemo<ComboboxOption[]>(() => {
        return selected.map((item) => ({value: getValue(item), label: getLabel(item)}));
    }, [selected, getValue, getLabel]);

    const itemsForValues = useCallback((values: string[]) => {
        return values.
            map((value) => itemsById.current.get(value)).
            filter((item): item is T => item !== undefined);
    }, []);

    return {options, selectedOptions, selectedValues, loading, error, onInputChange, itemsForValues};
}
