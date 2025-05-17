<script lang="ts" setup>
import { reactive } from 'vue'
import { Greet } from '../../wailsjs/go/main/App'

function selectFolder() {

}

</script>

<template>
    <main>
        <section>
            <div id="src-input" class="dir-select-container">
                <label for="src-dir">Source</label>
                <div class="dir-select">
                    <input id="src-dir" name="src-dir" autocomplete="off" type="text" />
                    <button @click="selectFolder">Select Folder</button>
                </div>
            </div>
            <div id="dst-input" class="dir-select-container">
                <label for="dst-dir">Destination</label>
                <div class="dir-select">
                    <input id="dst-dir" name="dst-dir" autocomplete="off" type="text" />
                    <button @click="selectFolder">Select Folder</button>
                </div>
            </div>
        </section>
    </main>
</template>

<style scoped>
main {
    width: 100%;
    display: flex;
}

section {
    margin: 0 auto;
    display: flex;
    gap: 3rem;
}

.dir-select-container {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
}

.dir-select {
    display: flex;
    gap: 0.25rem;
}
</style>
