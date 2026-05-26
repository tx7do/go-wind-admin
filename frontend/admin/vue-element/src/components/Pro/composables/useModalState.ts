import { ref, reactive } from "vue";

export function useModalState<T = any>(rowKey = "id") {
  const visible = ref(false);
  const mode = ref<"add" | "edit" | "view">("add");
  const formData = reactive<Record<string, any>>({});

  function open(m: typeof mode.value, row?: T) {
    mode.value = m;
    Object.keys(formData).forEach((k) => delete formData[k]);
    if (row) {
      Object.keys(row).forEach((k) => {
        formData[k] = (row as any)[k];
      });
    }
    visible.value = true;
  }

  return { visible, mode, formData, open, rowKey };
}
