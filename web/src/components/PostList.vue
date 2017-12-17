<template>
  <div>
    <section class="section">
      <h1 class="title">Story list</h1>
      <h2 class="subtitle">Select a story to start editing.</h2>
      <div class="columns is-multiline">
        <div class="column is-4" v-for="p in posts">
          <a class="box" v-on:click="editStory(p)">
            <p class="has-text-danger has-text-weight-semibold">{{ p.kicker }}</p>
            <p>{{ p.headline || p.snippet.name }}</p>
            <!-- <p>Last modified: {{ p.snippet.last_modified }}</p> -->
          </a>
        </div>
      </div>
      <p v-if="posts.length === 0">No posts.</p>
    </section>
    <section class="section">
      <h1 class="title">Upload</h1>
      <h2 class="subtitle">Choose an InDesign snippet to upload.</h2>
      <div class="file">
        <label class="file-label">
          <input class="file-input" type="file" name="snippet">
          <span class="file-cta">
            <span class="file-icon">
              <font-awesome-icon :icon="fileIcon" />
            </span>
            <span class="file-label">
              Choose a file...
            </span>
          </span>
        </label>
      </div>
    </section>
  </div>
</template>

<script>
import FontAwesomeIcon from '@fortawesome/vue-fontawesome'
import { faFile } from '@fortawesome/fontawesome-free-solid'
export default {
  name: 'PostEditor',
  data () {
    return {
      posts: []
    }
  },
  created () {
    fetch('http://127.0.0.1:8000/available-stories').then(response => {
      return response.json()
    }).then(posts => {
      this.posts = posts
    })
  },
  methods: {
    editStory (story) {
      this.$router.push({
        name: 'PostEditor',
        params: {
          story: story
        }
      })
    }
  },
  computed: {
    fileIcon () {
      return faFile
    }
  },
  components: {
    FontAwesomeIcon
  }
}
</script>

<style scoped>
</style>
