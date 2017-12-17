<template>
  <div>
    <section class="section">
      <div class="columns">
        <div class="column is-6">
          <h1 class="title">Post editor</h1>
          <h2 class="subtitle">Turn an InDesign snippet into a WordPress post.</h2>
        </div>
        <div class="column is-6">
          <div class="field is-grouped is-pulled-right">
            <p class="control">
              <a class="button is-primary" v-on:click="validate">Validate</a>
            </p>
            <p class="control">
              <a class="button is-primary is-pulled-right">Create post</a>
            </p>
          </div>
        </div>
      </div>
      <hr>
      <medium-editor class="has-text-danger is-uppercase" :text="story.kicker" :options="editorOptions" v-on:edit="editKicker" />
      <medium-editor class="title" :text="story.headline" :options="editorOptions" v-on:edit="editHeadline" />
      <medium-editor class="subtitle" :text="story.subdeck" :options="editorOptions" v-on:edit="editSubdeck" />
      <medium-editor class="has-text-weight-semibold" :text="story.authorName" :options="editorOptions" v-on:edit="editAuthorName" custom-tag="span" />,
      <medium-editor :text="story.authorTitle" :options="editorOptions" v-on:edit="editAuthorTitle" custom-tag="span" />
      <medium-editor :text="story.bodyText" :options="bodyTextEditorOptions" v-on:edit="editBodyText" />
    </section>
  </div>
</template>

<script>
import editor from 'vue2-medium-editor'
export default {
  name: 'PostEditor',
  data () {
    return {
      editorOptions: {
        disableReturn: true,
        toolbar: {
          buttons: ['italic']
        }
      },
      bodyTextEditorOptions: {
        toolbar: {
          buttons: ['italic', 'quote']
        }
      }
    }
  },
  methods: {
    validate () {
      if (this.story.kicker.endsWith('<br>')) {
        this.story.kicker = this.story.kicker.slice(0, -4)
      }
      if (this.story.headline.endsWith('<br>')) {
        this.story.headline = this.story.headline.slice(0, -4)
      }
      if (this.story.subdeck.endsWith('<br>')) {
        this.story.subdeck = this.story.subdeck.slice(0, -4)
      }
      if (this.story.authorName.endsWith('<br>')) {
        this.story.authorName = this.story.authorName.slice(0, -4)
      }
      if (this.story.authorTitle.endsWith('<br>')) {
        this.story.authorTitle = this.story.authorTitle.slice(0, -4)
      }
    },
    editKicker (operation) {
      this.story.kicker = operation.api.origElements.innerHTML
    },
    editHeadline (operation) {
      this.story.headline = operation.api.origElements.innerHTML
    },
    editSubdeck (operation) {
      this.story.subdeck = operation.api.origElements.innerHTML
    },
    editAuthorName (operation) {
      this.story.authorName = operation.api.origElements.innerHTML
    },
    editAuthorTitle (operation) {
      this.story.authorTitle = operation.api.origElements.innerHTML
    },
    editBodyText (operation) {
      this.story.bodyText = operation.api.origElements.innerHTML
    }
  },
  props: [
    'story'
  ],
  components: {
    'medium-editor': editor
  }
}
</script>

<style>
.medium-editor-element {
  outline: none;
}
.medium-editor-element p {
  margin-top: 20px;
}
</style>
