(function () {
  app.directive('datastoreFilter', function($rootScope){
    return {
      scope: {
        ds: '=',
        onChange: '&',
      },
      //restrict E means its can only be used as an element.
      restrict: 'E',
      templateUrl: function(elem, attr){
        return '/static/views/data/filterDirective.html?cache=1';
      },
      link: function($scope, elem, attrs) {
        // scope = either parent scope or its own child scope if scope set.
        // elem = jqLite wrapped element of: root object inside the template, so we can setup event handlers etc
      },
      controller: function($scope) {
        $scope.filters = [];

        $scope.addFilterInput = '';
        $scope.addFilterConditional = '==';
        $scope.addFilterColumn = '!!';

        $scope.getColName = function(colID){
          for (var i = 0; i < $scope.ds.Cols.length; i++){
            if ($scope.ds.Cols[i].UID == colID) return $scope.ds.Cols[i].Name;
          }
          return '?';
        }

        $scope.deleteFilter = function(index){
          $scope.filters.splice(index, 1);
          $scope.onChange({filters: $scope.filters});
        };

        $scope.addFilter = function(){
          if ($scope.addFilterColumn == '!!')return;

          if ($scope.addFilterInput.startsWith('{{')) {
            $scope.filters[$scope.filters.length] = {
              type: 'relativeConstraint',
              val: $scope.addFilterInput.slice(2,-2),
              col: $scope.addFilterColumn,
              conditional: $scope.addFilterConditional,
            }
          } else {
            $scope.filters[$scope.filters.length] = {
              type: 'literalConstraint',
              val: $scope.addFilterInput,
              col: $scope.addFilterColumn,
              conditional: $scope.addFilterConditional,
            }
          }
          $scope.addFilterInput = '';
          $scope.onChange({filters: $scope.filters});
        };


      },
    };
  });
})();
