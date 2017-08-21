(function () {

  function validationClass(value, shouldBeNum){
    if (value == undefined)return ['invalid'];
    if (value === '')return ['invalid'];
    if (shouldBeNum && isNaN(value))return ['invalid'];
    return ['valid'];
  }

    app.directive('datastoreEditModal', function($rootScope){
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
          return "/static/views/data/editModal.html?cachebust=3"
        },
        link: function($scope, elem, attrs) {
          // scope = either parent scope or its own child scope if scope set.
          // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
        },
        controller: function($scope) {
          $scope.open = false;
          $scope.isEditMode = false;
          $scope.ds = {};
          $scope.cols = [];
          $scope.cb = null;
          $scope.nameVldn = [];


          $rootScope.$on('create-datastore', function(event, args) {
            $scope.open = true;
            $scope.isEditMode = false;
            $scope.ds = {
              Name: '',
              Kind: 'DB',
            };
            $scope.cols = [];
            $scope.cb = args.cb;
          });

          $rootScope.$on('edit-datastore', function(event, args) {
            $scope.open = true;
            $scope.isEditMode = true;
            $scope.ds = args.ds;
            $scope.cols = args.ds.Cols;
            $scope.cb = args.cb;
          });



          $scope.newCol = function(type){
            $scope.cols[$scope.cols.length] = {Name: '', Datatype: type};
          }
          $scope.deleteCol = function(index){
            $scope.cols.splice(index, 1);
          }

          $scope.kind = function(k){
            switch (k){
              case 0:
                return "INT";
              case 2:
                return "FLOAT";
              case 3:
                return "STR";
              case 4:
                return "BLOB";
              case 5:
                return "TIME";
            }
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
              $scope.cb($scope.ds, $scope.cols);
              $scope.onCancel();
            }
          }

          //Checks and applies validation classes. Returns true if fields are valid.
          function valid(){
            $scope.nameVldn = [];
            $scope.nameVldn = validationClass($scope.ds.Name, false);
            for (var i = 0; i < $scope.cols.length; i++) {
              if (!$scope.cols[i].Name)return false;
            }
            return !$scope.nameVldn.includes('invalid');
          };

        }
      };
  });
})();
