<script lang="ts">
    interface Props {
        tasks?: any;
        title?: string;
    }

    let { tasks = [], title = "Flow" }: Props = $props();
</script>

<div class="bg-[var(--bg-color)] border border-[var(--border-color)] p-4 rounded-lg shadow-sm h-full">
    <h3 class="text-xl font-serif mb-4 pb-2 border-b border-[var(--border-color)] opacity-80 flex justify-between">
        <span>{title}</span>
        <span class="text-xs font-mono opacity-40 self-center">{tasks.length} ACTIVE</span>
    </h3>

    <div class="space-y-4">
        {#each tasks as task}
            <div class="flex items-start group p-2 rounded hover:bg-[var(--accent-color)]/5 transition-all cursor-pointer border-l-2 border-transparent hover:border-[var(--accent-color)]">
                <div class="flex-shrink-0 w-16 text-xs font-mono opacity-50 pt-1">
                    {task.time}
                </div>
                <div class="flex-1 ml-2">
                    <div class="font-medium opacity-90 group-hover:text-[var(--accent-color)] transition-colors">
                        {task.title}
                    </div>
                    {#if task.subtitle}
                        <div class="text-xs opacity-50 mt-0.5">{task.subtitle}</div>
                    {/if}
                </div>
                {#if task.color}
                    <div class="w-2 h-2 rounded-full mt-1.5" style="background-color: {task.color}"></div>
                {/if}
            </div>
        {/each}

        {#if tasks.length === 0}
            <div class="p-4 text-center opacity-40 italic font-serif text-sm">
                Flow is still. No tasks.
            </div>
        {/if}
    </div>
</div>
