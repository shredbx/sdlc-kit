import { writable, type Readable } from 'svelte/store';

export interface ScrollSpyOptions {
    headerOffset?: number;
    bottomMarginPercent?: number;
}

export interface ScrollSpyReturn {
    activeId: Readable<string | null>;
    spy: (node: HTMLElement, id?: string) => { destroy: () => void };
    destroy: () => void;
}

export function createScrollSpy(options: ScrollSpyOptions = {}): ScrollSpyReturn {
    const { headerOffset = 80, bottomMarginPercent = 60 } = options;
    const activeId = writable<string | null>(null);
    let lastActiveId: string | null = null;
    let observer: IntersectionObserver | null = null;
    const elements = new Map<Element, string>();

    function getObserver(): IntersectionObserver | null {
        if (typeof IntersectionObserver === 'undefined') return null;
        if (!observer) {
            observer = new IntersectionObserver(
                (entries) => {
                    for (const entry of entries) {
                        if (entry.isIntersecting) {
                            lastActiveId = elements.get(entry.target) ?? entry.target.id;
                            activeId.set(lastActiveId);
                        }
                    }
                    if (!entries.some(e => e.isIntersecting) && lastActiveId) {
                        activeId.set(lastActiveId);
                    }
                },
                { rootMargin: `-${headerOffset}px 0px -${bottomMarginPercent}% 0px`, threshold: 0 }
            );
        }
        return observer;
    }

    function spy(node: HTMLElement, id?: string) {
        const obs = getObserver();
        elements.set(node, id ?? node.id);
        obs?.observe(node);
        return {
            destroy() {
                obs?.unobserve(node);
                elements.delete(node);
            }
        };
    }

    function destroy() {
        observer?.disconnect();
        observer = null;
        elements.clear();
        lastActiveId = null;
    }

    return { activeId, spy, destroy };
}
