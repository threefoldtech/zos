import tfService from '../services/tfService'
/* eslint-disable */
export default ({
  state: {
    user: {},
    registeredNodes: [],
    nodes: undefined,
    registeredFarms: [],
    farms: [],
    nodeSpecs: {
      amountregisteredNodes: 0,
      amountregisteredFarms: 0,
      countries: 0,
      onlinenodes: 0,
      cru: 0,
      mru: 0,
      sru: 0,
      hru: 0,
      network: 0,
      volume: 0,
      container: 0,
      zdb_namespace: 0,
      k8s_vm: 0

    }
  },
  actions: {
    getName: async context => {
      var response = await tfService.getName()
      return response.data.name
    },
    getUser: async context => {
      var name = await context.dispatch('getName')
      var response = await tfService.getUser(name)
      context.commit('setUser', response.data)
    },
    getRegisteredNodes (context, params) {
      tfService.getNodes(undefined, params.size, params.page).then(response => {
        context.commit('setRegisteredNodes', response.data)
      })
    },
    getRegisteredNodesStats (context) {
      tfService.getNodeStats().then(response => {
        context.commit('setTotalSpecs', response.data)
      })
    },
    getRegisteredFarms (context, farmId) {
      tfService.registeredfarms(farmId).then(response => {
        context.commit('setAmountOfFarms', response.data)
        context.commit('setRegisteredFarms', response.data)
      })
    },
    getFarms: context => {
      tfService.getFarms(context.getters.user.id).then(response => {
        context.commit('setFarms', response.data)
      })
    },
    resetNodes: context => {
      context.commit('setNodes', undefined)
    }
  },
  mutations: {
    setRegisteredNodes (state, value) {
      state.registeredNodes = value
    },
    setRegisteredFarms (state, value) {
      state.registeredFarms = value
    },
    setFarms (state, value) {
      state.farms = value
    },
    setNodes (state, value) {
      state.nodes = value
    },
    setUser: (state, user) => {
      state.user = user
    },
    setAmountOfFarms (state, value) {
      state.nodeSpecs.amountregisteredFarms = value.length
    },
    setTotalSpecs (state, data) {
      debugger
      state.nodeSpecs.amountregisteredNodes = data.amountOfRegisteredNodes
      state.nodeSpecs.onlinenodes = data.onlineNodes
      state.nodeSpecs.countries = data.countries
      state.nodeSpecs.cru = data.totalCru
      state.nodeSpecs.mru = data.totalMru
      state.nodeSpecs.sru = data.totalSru
      state.nodeSpecs.hru = data.totalHru
      state.nodeSpecs.network = data.networks
      state.nodeSpecs.volume = data.volumes
      state.nodeSpecs.container = data.containers
      state.nodeSpecs.zdb_namespace = data.zdbs
      state.nodeSpecs.k8s_vm = data.k8s
    }
  },
  getters: {
    user: state => state.user,
    registeredNodes: state => state.registeredNodes,
    nodes: state => state.nodes,
    registeredFarms: state => state.registeredFarms,
    farms: state => state.farms,
    nodeSpecs: state => state.nodeSpecs
  }
})

function countOnlineNodes (data) {
  let onlinecounter = 0
  data.forEach(node => {
    const timestamp = new Date().getTime() / 1000
    const minutes = (timestamp - node.updated) / 60
    if (minutes < 20) onlinecounter++
  })
  return onlinecounter
}
