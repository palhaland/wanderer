<script lang="ts">
    import type { Snippet } from 'svelte';
    import { onMount } from 'svelte';

    interface Props {
        header?: Snippet;
        children?: Snippet;
        expanded?: boolean;
    }

    let { header, children, expanded = $bindable(false) }: Props = $props();
    let isMobile = $state(false);

    onMount(() => {
        const updateMedia = () => isMobile = window.innerWidth < 768;
        updateMedia();
        window.addEventListener('resize', updateMedia);
        return () => window.removeEventListener('resize', updateMedia);
    });

    function toggle() {
        expanded = !expanded;
    }
</script>

<div 
    class="md:relative md:h-full md:w-full fixed bottom-0 left-0 right-0 z-[100] bg-background md:bg-transparent border-t md:border-t-0 border-input-border rounded-t-3xl md:rounded-t-none shadow-[0_-8px_30px_rgb(0,0,0,0.12)] md:shadow-none transition-all duration-300 ease-in-out flex flex-col pointer-events-auto overflow-hidden"
    style="height: {isMobile ? (expanded ? '80dvh' : '80px') : '100%'};"
>
    <!-- Mobile: Drag Handle / Header Toggle -->
    <div 
        class="md:hidden flex flex-col items-center py-3 cursor-pointer select-none shrink-0" 
        onclick={toggle}
        role="button"
        tabindex="0"
        onkeydown={(e) => e.key === 'Enter' && toggle()}
    >
        <div class="w-12 h-1.5 bg-gray-300 rounded-full mb-2"></div>
        {#if header}
            <div class="w-full px-4 text-center">
                {@render header()}
            </div>
        {/if}
    </div>

    <!-- Scrollable Content -->
    <div class="flex-1 overflow-y-auto px-0 md:px-0 pb-8 md:pb-0" class:hidden_mobile={!expanded}>
        {@render children?.()}
    </div>
</div>

<style>
    .hidden_mobile {
        @media (max-width: 767px) {
            display: none;
        }
    }
</style>
