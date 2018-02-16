(function () {

    app.directive('fsActionsModal', function($rootScope, $sce, $http){
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
          return "/static/views/fs/actionsModal.html?cachebust=1"
        },
        link: function($scope, elem, attrs) {
          // scope = either parent scope or its own child scope if scope set.
          // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
        },
        controller: function($scope) {
          $scope.open = false;
          $scope.loading = false;
          $scope.path = '';
          $scope.error = null;

          $rootScope.$on('fs-actions-modal', function(event, args) {
            $scope.open = true;
            $scope.path = args.path;
            $scope.actions = [];
            $scope.data = {};
            $scope.getActions().then(function(result){
              $scope.actions = result.data;
              console.log(result);
            })
          });


          $scope.getActions = function(f){
              $scope.loading = true;
              return $http({
                method: 'POST',
                url: '/web/v1/fs/actions',
                data: {path: $scope.path},
              }).then(function(result){
                $scope.loading = false;
                return result;
              });
          }

          $scope.run = function(action) {
            switch (action.kind) {
              case 'button':
                $scope.doRunAction({path: $scope.path, id: action.ID}).then(function(result){
                  //action.result = result.data;
                  $scope.actions[$scope.actions.indexOf(action)].results = result.data;
                  console.log("action result: ", result);
                })
                break;
              case '1_string':
                $scope.doRunAction({path: $scope.path, id: action.ID, payload: $scope.data[action.ID]}).then(function(result){
                  //action.result = result.data;
                  $scope.actions[$scope.actions.indexOf(action)].results = result.data;
                  console.log("action result: ", result);
                })
            }
          }

          $scope.doRunAction = function(payload) {
            $scope.loading = true;
            return $http({
              method: 'POST',
              url: '/web/v1/fs/runAction',
              data: payload,
            }).then(function(result){
              $scope.loading = false;
              if (result.data.success === false) {
                $scope.error = result.data;
              }
              return result;
            });
          }

          $scope.close = function(){
            $scope.open = false;
          }
        }
      };
  });
})();
