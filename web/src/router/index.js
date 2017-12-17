import Vue from 'vue'
import Router from 'vue-router'
import StoryEditor from '@/components/StoryEditor'
import StoryList from '@/components/StoryList'

Vue.use(Router)

export default new Router({
  mode: 'history',
  routes: [
    {
      path: '/',
      name: 'StoryList',
      component: StoryList
    },
    {
      path: '/editor',
      name: 'StoryEditor',
      component: StoryEditor,
      props: true
    }
  ]
})
