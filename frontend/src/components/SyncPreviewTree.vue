<script lang="ts" setup>
import { computed, PropType } from 'vue';
import { lsync } from '../../wailsjs/go/models';

const props = defineProps({
    dirPath: {
        type: String,
        required: true
    },
    dirName: {
        type: String,
        required: true
    },
    dirPreview: {
        type: Object as PropType<lsync.SyncPreview>,
        required: true
    },
    ignoreDeleted: {
        type: Boolean,
        required: true
    }
})

const subdirs = computed(() => Object.entries(props.dirPreview.Subdirs).map(kv => {
    return {
        name: kv[0],
        path: pathJoin(props.dirPath, kv[0]),
        preview: kv[1]
    };
}));

const files = computed(() => Object.entries(props.dirPreview.Files).map(kv => {
    return {
        name: kv[0],
        class: getClassFromSyncStatus(kv[1]),
        key: pathJoin(props.dirPath, kv[0])
    };
}));

function getClassFromSyncStatus(status: string): string {
    if (!status) {
        return ""
    }
    switch (status) {
        case lsync.SyncStatus.Created:
            return "created";
        case lsync.SyncStatus.Modified:
            return "modified";
        case lsync.SyncStatus.Deleted:
            return props.ignoreDeleted ? "ignored" : "deleted";
    }
    return ""
}

function pathJoin(path: string, name: string): string {
    return `${path}\\${name}`;
}
</script>

<!-- Need to add key for v-for -->
<template>
    <ul>
        <li>
            <details open>
                <summary :class="getClassFromSyncStatus(dirPreview.Status)">{{ dirName }}</summary>
                <div class="list-children">
                    <SyncPreviewTree v-for="v in subdirs" :key="v.path" :dir-path="v.path" :dir-name="v.name"
                        :dir-preview="v.preview" :ignore-deleted="props.ignoreDeleted">
                    </SyncPreviewTree>
                    <ul v-if="dirPreview && Object.keys(dirPreview.Files).length > 0">
                        <li v-for="v in files" :class="v.class" :key="v.key">{{ v.name }}
                        </li>
                    </ul>
                </div>
            </details>
        </li>
    </ul>
</template>

<style scoped>
ul {
    flex: 1;
    list-style: none;
    padding-left: 0;
    margin: 0.5rem 0;
}

summary {
    cursor: pointer;
}

li {
    text-wrap-mode: nowrap;
}

.list-children {
    padding-left: 1rem;
}

.modified {
    color: #ffbf00;
}

.created {
    color: #00a515;
}

.deleted {
    color: #d20000;
}

.ignored {
    color: #a0a0a0;
}
</style>