(function () {

  function validationClass(value, shouldBeNum){
    if (value == undefined)return ['invalid'];
    if (value === '')return ['invalid'];
    if (shouldBeNum && isNaN(value))return ['invalid'];
    return ['valid'];
  }

    app.directive('integrationEditModal', function($rootScope){
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
          return "/static/views/integration/editModal.html?cachebust=6"
        },
        link: function($scope, elem, attrs) {
          // scope = either parent scope or its own child scope if scope set.
          // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
        },
        controller: function($scope) {
          $scope.open = false;
          $scope.isEditMode = false;
          $scope.integration = {};
          $scope.triggers = [];
          $scope.cb = null;
          $scope.nameVldn = [];


          $rootScope.$on('create-integration', function(event, args) {
            $scope.open = true;
            $scope.isEditMode = false;
            $scope.integration = {
              Name: '',
              Kind: 'Runnable',
            };
            $scope.triggers = [];
            $scope.cb = args.cb;
          });

          $rootScope.$on('edit-integration', function(event, args) {
            $scope.open = true;
            $scope.isEditMode = true;
            $scope.integration = args.integration;
            $scope.triggers = args.integration.Triggers || [];
            $scope.cb = args.cb;
          });



          $scope.newTrigger = function(type){
            $scope.triggers[$scope.triggers.length] = {Name: '', Kind: type};
          }
          $scope.deleteTrigger = function(index){
            $scope.triggers.splice(index, 1);
          }

          $scope.onCancel = function(){
            $scope.open = false;
            $scope.isEditMode = false;
            $scope.ds = {};
            $scope.cols = [];
            $scope.cb = null;
            $scope.nameVldn = '';
          }

          $scope.onFinish = function(){
            if (valid()){
              $scope.cb($scope.integration, $scope.triggers);
              $scope.onCancel();
            }
          }

          //Checks and applies validation classes. Returns true if fields are valid.
          function valid(){
            $scope.nameVldn = [];
            $scope.nameVldn = validationClass($scope.integration.Name, false);
            for (var i = 0; i < $scope.triggers.length; i++) {
              if (!$scope.triggers[i].Name)return false;
              if ($scope.triggers[i].Kind == "CRON" || $scope.triggers[i].Kind == "HTTP" || $scope.triggers[i].Kind == "PUBSUB") {
                if (!$scope.triggers[i].Val1)return false;
              }
              if ($scope.triggers[i].Kind == "PUBSUB" && !$scope.triggers[i].Val2)
                return false;
              if ($scope.triggers[i].Kind == "PUBSUB" && !/^projects\/[^\/]+\/topics\/[^\/]+$/.test($scope.triggers[i].Val1)) {
                $scope.triggers[i].subVldn = ['invalid'];
                return false;
              } else {
                $scope.triggers[i].subVldn = [];
              }
            }
            return !$scope.nameVldn.includes('invalid');
          };

        }
      };
  });
})();
