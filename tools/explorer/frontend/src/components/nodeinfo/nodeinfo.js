export default {
  name: 'nodeinfo',
  props: ['node'],
  data () {
    return {
    }
  },
  mounted () {
    console.log(this.node)
  },
  methods: {
    getPercentage (type) {
      return (this.node.reservedResources[type] / this.node.totalResources[type]) * 100
    }
  }
}
