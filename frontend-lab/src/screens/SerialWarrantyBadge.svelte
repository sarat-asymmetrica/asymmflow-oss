<script lang="ts">
  /* L4 cell ejection (descriptor.ts's ColumnSpec.cell): DataTable's built-in
   * Badge rendering is tied to ONE StatusSpec per table (its `status` prop),
   * already spent here on the lifecycle "Stage" column. Warranty needs its
   * own, independent Badge dimension, so this column steps outside the
   * engine explicitly instead of fighting it. */
  import Badge from '$kernel/controls/Badge.svelte'
  import type { SerialTraceRow } from '../bridge/serial-trace'
  import { warrantyLabel, warrantyTone } from './serial-trace.svelte'

  let { row }: { row: SerialTraceRow } = $props()
</script>

<Badge tone={warrantyTone(row)} label={warrantyLabel(row)} />
