<template>
  <v-app dark>
    <v-navigation-drawer mini-variant app class="primary rounded" dark>
      <v-layout column fill-height justify-end>
        <div>
          <v-toolbar color="secondary darken-2 " class="py-3">
            <v-badge bottom right overlap color="primary">
              <template v-slot:badge>
                <v-icon size="12" dark>{{$route.meta.icon}}</v-icon>
              </template>
              <!--slot can be any component-->
              <v-avatar>
                <v-img src="./assets/logo.jpg" />
              </v-avatar>
            </v-badge>
          </v-toolbar>
        </div>
        <div>
          <v-list-item
            v-for="(route, i) in routes.filter(r => r.meta.position == 'top')"
            :key="i"
            :to="route"
            active-class="secondary--text"
          >
            <v-list-item-icon>
              <v-icon>{{ route.meta.icon }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content>
              <v-list-item-title class="title text-capitalize">{{route.meta.displayName}}</v-list-item-title>
            </v-list-item-content>
          </v-list-item>
        </div>
        <v-spacer></v-spacer>
        <div>
          <v-list-item
            v-for="(route, i) in routes.filter(r => r.meta.position == 'bottom')"
            :key="i"
            :to="route"
            active-class="secondary--text"
          >
            <v-list-item-icon>
              <v-icon>{{ route.meta.icon }}</v-icon>
            </v-list-item-icon>
            <v-list-item-content>
              <v-list-item-title class="title text-capitalize">{{route.meta.displayName}}</v-list-item-title>
            </v-list-item-content>
          </v-list-item>
        </div>
      </v-layout>
    </v-navigation-drawer>

    <v-content class="content">
      <v-col>
        <v-row class="pa-4 mx-1">
          <h1 class="headline pt-0 pb-1 text-uppercase">
            <span>TF</span>
            <span class="font-weight-light">explorer</span>
            <span class="title font-weight-light">- {{$route.meta.displayName}}</span>
          </h1>
          <v-progress-circular
            class="refresh"
            v-if="nodePage || farmPage"
            indeterminate
            color="primary"
          ></v-progress-circular>
          <v-btn class="refresh" icon v-else @click="refresh">
            <v-icon
              big
              color="primary"
              left
            >
              fas fa-sync-alt
            </v-icon>
          </v-btn>
        </v-row>
        <router-view></router-view>
      </v-col>
    </v-content>
    <v-bottom-navigation
      v-if="$vuetify.breakpoint.mdAndDown"
      grow
      dark
      class="primary topround"
      app
      fixed
      shift
      :value="$route.name"
    >
      <v-btn
        :value="route.name"
        icon
        v-for="(route, i) in routes"
        :key="i"
        @click="$router.push(route)"
      >
        <span>{{route.meta.displayName}}</span>
        <v-icon>{{route.meta.icon}}</v-icon>
      </v-btn>
    </v-bottom-navigation>
  </v-app>
</template>

<script>
import { mapGetters, mapActions } from 'vuex'

export default {
  name: 'App',
  components: {},
  data: () => ({
    showDialog: false,
    dilogTitle: 'title',
    dialogBody: '',
    dialogActions: [],
    dialogImage: null,
    block: null,
    showBadge: true,
    menu: false,
    start: undefined,
    refreshInterval: undefined
  }),
  computed: {
    routes () {
      return this.$router.options.routes
    },
    ...mapGetters([
      'nodePage',
      'farmPage'
    ])
  },
  mounted () {
    // keep track when the user opened this app
    this.start = new Date()

    // refresh every 10 minutes
    this.refreshInterval = setInterval(this.refreshData(), 60000)

    // if user loses focus, clear the refreshing interval
    // we don't refresh data if the page is not focused.
    window.onblur = () => {
      clearInterval(this.refreshInterval)
    }

    // instead of refreshing every 10 minutes in the background
    // we do following:
    // when the user enters the page again and 10 minutes have passed since the first visit
    // refresh all data. Start the refresh interval again (since we assume the user is going to stay on this page)
    // if the user decides to leave this page again this interval will be cleared again
    window.onfocus = () => {
      const now = new Date()
      let elapsedTime = now - this.start
      // strip the ms
      elapsedTime /= 1000
      const seconds = Math.round(elapsedTime)

      // if 10 minutes are passed since last focus, refresh data.
      if (seconds >= 600) {
        this.start = new Date()
        this.refreshData()
        this.refreshInterval = setInterval(this.refreshData(), 60000)
      }
    }
  },
  methods: {
    ...mapActions(['refreshData']),
    refresh () {
      this.refreshData()
    }
  }
}
</script>

<style lang="scss">
.content {
  background: #fafafa !important;
}
.topround {
  border-radius: 10px 10px 0 0 !important;
}
.rounded {
  border-radius: 0 10px 10px 0 !important;
}
.v-menu__content,
.v-card {
  border-radius: 10px !important;
}
.v-card__title {
  font-size: 18px !important;
}
.spinner {
  margin-left: 20px;
}
.refresh {
  position: absolute !important;
  right: 25px !important;
  top: 30px !important;
  font-size: 30px !important;
}
</style>
