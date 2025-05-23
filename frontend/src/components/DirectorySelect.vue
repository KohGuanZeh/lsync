<script lang="ts">
export enum DirType {
    Src = "src",
    Dst = "dst"
}

function getLabel(dirType: DirType): string {
    if (dirType == DirType.Dst) {
        return "Destination";
    }
    return "Source";
}
</script>


<script lang="ts" setup>
import { PropType, watch } from 'vue';
import { SelectDirectory } from '../../wailsjs/go/backend/App';

const props = defineProps({
    dirType: {
        type: String as PropType<DirType>,
        required: true,
        default: DirType.Src,
    },
    dirVal: {
        type: String,
        required: true,
        default: ""
    }
});
const emit = defineEmits<{ (e: 'update-dir-value', dirType: DirType, newDirVal: string): void }>();
const label = getLabel(props.dirType);

watch(() => props.dirVal, (newDirVal) => {
    emit('update-dir-value', props.dirType, newDirVal);
});

function selectDirectory() {
    const title = `Select ${label} Directory`;
    SelectDirectory(title).then(res => {
        if (res.length != 0) {
            emit('update-dir-value', props.dirType, res);
        }
    }, err => {
        console.log(err);
    });
}
</script>

<template>
    <div class="dir-select-container">
        <label :for="`${dirType}-dir`">{{ label }}</label>
        <div class="dir-select-controls">
            <input :name="`${dirType}-dir`" v-model="props.dirVal" :placeholder="`Select ${label} Directory...`"
                class="dir-select-input" autocomplete="off" type="text" />
            <button class="dir-select-btn" @click="selectDirectory">
                <i class="fa-solid fa-folder"></i>
            </button>
        </div>
    </div>
</template>

<style scoped>
.dir-select-container {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
}

.dir-select-controls {
    display: flex;
}

.dir-select-input {
    flex: 1;
    border-radius: 0.25rem 0 0 0.25rem;
}

.dir-select-btn {
    border-radius: 0 0.25rem 0.25rem 0;
    padding: 0.5rem 1rem;
    margin: auto;
}
</style>