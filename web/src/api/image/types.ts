export interface ImagesRequest {
  page: number;
  size: number;
}

export interface ImagesResponse {
  code: number;
  msg: string;
  data: ImagePaginate;
}

export interface ImagePaginate{
  total: number;
  list: string[];
}

export interface ImageRequest{

}