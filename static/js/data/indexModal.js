(function () {

  function validationClass(value, shouldBeNum){
    if (value == undefined)return ['invalid'];
    if (value === '')return ['invalid'];
    if (shouldBeNum && isNaN(value))return ['invalid'];
    return ['valid'];
  }

    app.directive('datastoreIndexModal', function($rootScope, $sce, $http){
      return {
        //scope allows us to setup variable bindings to the parent scope. By default, we share the parent scope. For an isolated one, we should
        //pass an object as the scope attribute which has a dict of the variable name for us, and a string describing where and how to bind it.
        //scope: {color: '@colorAttr'} binds 'color' to the value of color-attr with a one way binding. Only strings supported here: color-attr="{{color}}"
        //scope: {color: '=color'} binds 'color' two way. Pass the object here: color="color".
        scope: {

        },
        //restrict E means its can only be used as an element.
        restrict: 'E',
        templateUrl: function(elem, attr){
          return "/static/views/data/indexModal.html?cachebust=1"
        },
        link: function($scope, elem, attrs) {
          // scope = either parent scope or its own child scope if scope set.
          // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
        },
        controller: function($scope) {
          $scope.open = false;
          $scope.ds = {};
          $scope.nameVldn = [];
          $scope.indexes = [];

          $scope.getIndexes = function(){
              $scope.loading = true;
              return $http({
                method: 'POST',
                url: '/web/v1/data/indexes/get',
                data: {UID: $scope.ds.UID},
              }).then(function(result){
                $scope.loading = false;
                return result;
              }, function(e){
                $scope.loading = false;
                $scope.error = e;
                return [];
              });
          }

          $rootScope.$on('datastore-indexes', function(event, args) {
            $scope.open = true;
            $scope.ds = args.ds;
            $scope.indexes = [];
            $scope.new_index_name = '';
            $scope.new_index_cols = '';
            $scope.getIndexes().then(function(response){
              $scope.indexes = response.data;
            })
          });

          $scope.onDone = function(){
            $scope.ds = {};
            $scope.open = false;
            $scope.error = undefined;
          }

          $scope.onDelete = function(uid){
            $scope.loading = true;
            $http({
              method: 'POST',
              url: '/web/v1/data/indexes/delete',
              data: {UID: uid},
            }).then(function(result){
              $scope.getIndexes().then(function(response){
                $scope.indexes = response.data;
              })
            }, function(e){
              $scope.loading = false;
              $scope.error = e;
            });
          }

          $scope.onNew = function(){
            $scope.loading = true;
            $http({
              method: 'POST',
              url: '/web/v1/data/indexes/new',
              data: {UID: $scope.ds.UID, Name: $scope.new_index_name, Cols: $scope.new_index_cols.split(',').map(s => s.trim())},
            }).then(function(result){
              $scope.getIndexes().then(function(response){
                $scope.indexes = response.data;
              })
            }, function(e){
              $scope.loading = false;
              $scope.error = e;
            });
          }

          //Checks and applies validation classes. Returns true if fields are valid.
          function valid(){
            $scope.nameVldn = [];
            $scope.nameVldn = validationClass($scope.ds.Name, false);
            return !$scope.nameVldn.includes('invalid');
          };

        }
      };
  });
})();
